from flask import Blueprint, render_template, jsonify, request
from .utils import execute_query
from datetime import datetime, timedelta
import random

routes = Blueprint('routes', __name__)

@routes.route("/")
def home():
    return render_template("dashboard.html")

@routes.route("/daily/data/<tab>")
def get_daily_data(tab):
    start_date = request.args.get('start_date')
    end_date = request.args.get('end_date')

    if not start_date or not end_date:
        return jsonify({"error": "Please provide start_date and end_date"}), 400

    query = {
        "calls": """
            SELECT DATE(c.enter_queue_date) AS report_date, COUNT(*) AS total_calls
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN %s AND %s
              AND (c.type = 'in' OR c.type = 'abandon')
            GROUP BY report_date
            ORDER BY report_date;
        """,
        "aht": """
            SELECT DATE(c.enter_queue_date) AS report_date, ROUND(AVG(c.call_duration), 2) AS avg_call_duration
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN %s AND %s
            GROUP BY report_date
            ORDER BY report_date;
        """,
        "chats": """
            SELECT DATE(c.assign_date) AS report_date, COUNT(*) AS total_chats
            FROM chat_report c
            WHERE DATE(c.assign_date) BETWEEN %s AND %s
            GROUP BY report_date
            ORDER BY report_date;
        """,
        "frt": """
            SELECT DATE(c.assign_date) AS report_date, ROUND(AVG(c.chat_frt), 2) AS avg_chat_frt
            FROM chat_report c
            WHERE DATE(c.assign_date) BETWEEN %s AND %s
            GROUP BY report_date
            ORDER BY report_date;
        """,
        "sl": """
            SELECT DATE(c.enter_queue_date) AS report_date,
                   ROUND(SUM(CASE WHEN c.queue_wait_time <= 20 THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) AS sl
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN %s AND %s
              AND c.type = 'in'
              AND c.queue_name = 'm10'
            GROUP BY report_date
            ORDER BY report_date;
        """,
        "rt": """
            SELECT DATE(c.assign_date) AS report_date, ROUND(AVG(c.resolution_time_avg), 2) AS resolution_time_avg
            FROM chat_report c
            WHERE DATE(c.assign_date) BETWEEN %s AND %s
            GROUP BY report_date
            ORDER BY report_date;
        """
    }.get(tab)

    if not query:
        return jsonify({"error": "Invalid tab name"}), 400

    data = execute_query(query, (start_date, end_date))
    return jsonify(data)

@routes.route("/hourly/data/<tab>")
def get_hourly_data(tab):
    start_date = request.args.get('start_date')
    end_date = request.args.get('end_date')

    if not start_date or not end_date:
        return jsonify({"error": "Please provide start_date and end_date"}), 400

    query = {
        "calls": """
            SELECT HOUR(c.enter_queue_date) AS hour, COUNT(*) AS total_calls
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN %s AND %s
              AND (c.type = 'in' OR c.type = 'abandon')
            GROUP BY hour
            ORDER BY hour;
        """,
        "aht": """
            SELECT HOUR(answer_date) AS hour, ROUND(AVG(call_duration), 2) AS avg_call_duration
            FROM call_report
            WHERE DATE(answer_date) BETWEEN %s AND %s
            GROUP BY hour
            ORDER BY hour;
        """,
        "chats": """
            SELECT HOUR(c.assign_date) AS hour, COUNT(*) AS total_chats
            FROM chat_report c
            WHERE DATE(c.assign_date) BETWEEN %s AND %s
            GROUP BY hour
            ORDER BY hour;
        """,
        "frt": """
            SELECT HOUR(assign_date) AS hour, ROUND(AVG(chat_frt), 2) AS avg_chat_frt
            FROM chat_report
            WHERE DATE(assign_date) BETWEEN %s AND %s
            GROUP BY hour
            ORDER BY hour;
        """,
        "sl": """
            SELECT HOUR(enter_queue_date) AS hour, ROUND(SUM(queue_wait_time <= 20) / COUNT(*) * 100, 2) AS sl
            FROM call_report c
            WHERE DATE(enter_queue_date) BETWEEN %s AND %s AND type = 'in' AND queue_name = 'm10'
            GROUP BY hour
            ORDER BY hour;
        """,
        "rt": """
            SELECT HOUR(assign_date) AS hour, ROUND(AVG(resolution_time_avg), 2) AS resolution_time_avg
            FROM chat_report c
            WHERE DATE(assign_date) BETWEEN %s AND %s
            GROUP BY hour
            ORDER BY hour;
        """
    }.get(tab)

    if not query:
        return jsonify({"error": "Invalid tab name"}), 400

    data = execute_query(query, (start_date, end_date))
    return jsonify(data)

@routes.route('/forecast/daily/data')
def forecast_daily_data():
    """Get daily forecast data for a period"""
    start_date = request.args.get('start_date')
    end_date = request.args.get('end_date')
    mode = request.args.get('mode', 'optimal')
    
    # This is a mock of what your backend should return
    # In production, replace this with your actual forecasting logic
    
    # Mock historical stats
    historical_stats = {
        "calls": {"avg": 550, "min": 420, "max": 780},
        "chats": {"avg": 320, "min": 240, "max": 480},
        "sl": {"avg": 92.4, "min": 82, "max": 97},
        "aht": {"avg": 380, "min": 320, "max": 450},
    }
    
    # Generate mock forecast data for each day in the range
    forecast = {}
    
    start = datetime.strptime(start_date, '%Y-%m-%d')
    end = datetime.strptime(end_date, '%Y-%m-%d')
    date_range = [start + timedelta(days=i) for i in range((end - start).days + 1)]
    
    for date in date_range:
        date_str = date.strftime('%Y-%m-%d')
        
        # Adjust values based on mode
        mode_factor = {
            'optimal': 1.0,
            'aggressive': 0.85,  # Better performance targets
            'conservative': 1.15  # More conservative estimates
        }.get(mode, 1.0)
        
        # Daily variations - weekends have different patterns
        is_weekend = date.weekday() >= 5
        weekend_factor = 0.7 if is_weekend else 1.0
        
        daily_calls = int(historical_stats["calls"]["avg"] * weekend_factor * mode_factor * (0.9 + random.random() * 0.2))
        daily_chats = int(historical_stats["chats"]["avg"] * weekend_factor * mode_factor * (0.9 + random.random() * 0.2))
        daily_sl = min(99, max(80, historical_stats["sl"]["avg"] * (2 - mode_factor) * (0.95 + random.random() * 0.1)))
        daily_aht = historical_stats["aht"]["avg"] * mode_factor * (0.95 + random.random() * 0.1)
        
        forecast[date_str] = {
            "daily_calls": daily_calls,
            "daily_chats": daily_chats,
            "daily_sl": round(daily_sl, 1),
            "daily_aht": round(daily_aht),
            "daily_abandoned": int(daily_calls * (1 - daily_sl/100) * 0.8),
            "daily_frt": round(300 + random.random() * 120),  # FRT in seconds
            "daily_rt": round(600 + random.random() * 240),   # RT in seconds
            "daily_required_agents": int((daily_calls + daily_chats) / 45 * mode_factor),
            "calls_comment": f"{'Lower' if is_weekend else 'Normal'} volume expected",
            "chats_comment": f"{'Weekend' if is_weekend else 'Weekday'} pattern applied",
            "sl_comment": f"{'Target achievable' if daily_sl >= 95 else 'Below target, needs attention'}",
            "aht_comment": f"Based on {'weekend' if is_weekend else 'weekday'} historical data",
            "agents_comment": f"For target SL of 95%"
        }
    
    return jsonify({
        "historical_stats": historical_stats,
        "forecast": forecast
    })