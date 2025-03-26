from flask import jsonify, request
from datetime import datetime, timedelta
import numpy as np
from scipy import stats
import pandas as pd
from sklearn.ensemble import IsolationForest

def calculate_optimal_value(historical_data, metric_type):
    """Вычисляет оптимальное значение метрики, удаляя выбросы"""
    if not historical_data:
        return 0
        
    # Удаляем выбросы с помощью IsolationForest
    iso = IsolationForest(contamination=0.1)
    yhat = iso.fit_predict(np.array(historical_data).reshape(-1, 1))
    mask = yhat != -1
    clean_data = [x for x, m in zip(historical_data, mask) if m]
    
    if not clean_data:
        return np.mean(historical_data)

    # Для разных метрик используем разные методы
    if metric_type in ['calls', 'chats']:
        # Берём 75-й перцентиль для объёмов
        return np.percentile(clean_data, 75)
    elif metric_type == 'sl':
        # Для SL берём значения выше 95%
        good_sl = [x for x in clean_data if x >= 95]
        return np.mean(good_sl) if good_sl else 95
    elif metric_type in ['aht', 'frt', 'rt']:
        # Для временных метрик берём нижний квартиль
        return np.percentile(clean_data, 25)
    else:
        return np.mean(clean_data)

def get_historical_pattern(data, metric):
    """Определяет паттерны по дням недели"""
    df = pd.DataFrame(data)
    df['day_of_week'] = pd.to_datetime(df['report_date']).dt.dayofweek
    patterns = df.groupby('day_of_week')[metric].mean().to_dict()
    return patterns

@app.route("/forecast/data")
def get_forecast():
    start_date = request.args.get('start_date')
    end_date = request.args.get('end_date')
    mode = request.args.get('mode', 'optimal')  # optimal, aggressive, conservative
    
    # Получаем исторические данные за последние 30 дней
    historical_end = datetime.strptime(start_date, '%Y-%m-%d')
    historical_start = historical_end - timedelta(days=30)
    
    # Получаем исторические данные
    metrics_data = {
        'calls': [],
        'aht': [],
        'sl': [],
        'chats': [],
        'frt': [],
        'rt': [],
        'abandoned': []
    }
    
    # Заполняем исторические данные из БД...
    
    forecast = {}
    current_date = datetime.strptime(start_date, '%Y-%m-%d')
    end_datetime = datetime.strptime(end_date, '%Y-%m-%d')
    
    while current_date <= end_datetime:
        date_str = current_date.strftime('%Y-%m-%d')
        day_of_week = current_date.weekday()
        
        forecast[date_str] = {}
        
        for hour in range(24):
            hour_data = {}
            
            for metric, data in metrics_data.items():
                # Фильтруем данные для конкретного часа
                hour_values = [x for x, h in data if h == hour]
                
                if mode == 'optimal':
                    value = calculate_optimal_value(hour_values, metric)
                elif mode == 'aggressive':
                    if metric == 'sl':
                        value = 95  # Целевой SL
                    elif metric in ['aht', 'frt', 'rt']:
                        value = min(hour_values) if hour_values else 0  # Лучшее время
                    else:
                        value = np.percentile(hour_values, 85) if hour_values else 0
                else:  # conservative
                    patterns = get_historical_pattern(data, metric)
                    avg_value = np.mean(hour_values) if hour_values else 0
                    day_factor = patterns.get(day_of_week, 1)
                    value = avg_value * day_factor
                
                hour_data[metric] = round(value)
            
            # Рассчитываем требуемое количество агентов
            total_contacts = hour_data['calls'] + hour_data['chats']
            hour_data['required_agents'] = max(1, round(total_contacts / 15))  # Предполагаем 15 контактов на агента
            
            forecast[date_str][hour] = hour_data
            
        current_date += timedelta(days=1)
    
    return jsonify({
        'forecast': forecast,
        'historical_stats': {
            metric: {
                'avg': np.mean([x for x, _ in data]),
                'max': max([x for x, _ in data]),
                'min': min([x for x, _ in data])
            } for metric, data in metrics_data.items() if data
        }
    })
