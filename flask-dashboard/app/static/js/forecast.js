// Enhanced forecast.js with better algorithms and visualization

// Constants and configurations
const FORECAST_TYPES = {
    CONSERVATIVE: 'conservative',
    OPTIMAL: 'optimal',
    AGGRESSIVE: 'aggressive'
};

// Main forecasting module
const Forecaster = {
    // Cache for historical data
    historicalData: null,
    
    // Initialize forecast functionality
    init: function() {
        // Event listeners for forecast type selection
        document.querySelectorAll('.forecast-type-selector').forEach(el => {
            el.addEventListener('click', function(e) {
                e.preventDefault();
                const forecastType = this.getAttribute('data-type');
                Forecaster.updateForecastView(forecastType);
            });
        });
        
        // Initialize datepickers with appropriate defaults
        this.initDateRanges();
    },
    
    // Set up date ranges for forecasting (next 7 days default)
    initDateRanges: function() {
        const today = new Date();
        const nextWeek = new Date();
        nextWeek.setDate(nextWeek.getDate() + 7);
        
        // Format for input date fields
        const formatDate = (date) => {
            const year = date.getFullYear();
            const month = String(date.getMonth() + 1).padStart(2, '0');
            const day = String(date.getDate()).padStart(2, '0');
            return `${year}-${month}-${day}`;
        };
        
        // Set default values if available
        if (document.getElementById('forecast-start-date')) {
            document.getElementById('forecast-start-date').value = formatDate(today);
        }
        
        if (document.getElementById('forecast-end-date')) {
            document.getElementById('forecast-end-date').value = formatDate(nextWeek);
        }
    },
    
    // Load forecast data
    loadForecast: function(startDate, endDate, mode, isDaily = false) {
        const endpoint = isDaily ? '/forecast/daily/data' : '/forecast/data';
        const params = {
            start_date: startDate,
            end_date: endDate,
            mode: mode || FORECAST_TYPES.OPTIMAL
        };
        
        const queryString = Object.keys(params)
            .map(key => `${key}=${encodeURIComponent(params[key])}`)
            .join('&');
            
        return fetch(`${endpoint}?${queryString}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error ${response.status}`);
                }
                return response.json();
            });
    },
    
    // Update the forecast visualization based on selected type
    updateForecastView: function(forecastType) {
        const startDate = document.getElementById('forecast-start-date').value;
        const endDate = document.getElementById('forecast-end-date').value;
        const isDaily = document.querySelector('.forecast-view-toggle.active')?.getAttribute('data-view') === 'daily';
        
        // Show loading indicator
        document.getElementById('forecast-container').innerHTML = '<div class="loading">Loading forecast data...</div>';
        
        this.loadForecast(startDate, endDate, forecastType, isDaily)
            .then(data => {
                // Store historical data for comparative analysis
                this.historicalData = data.historical_stats;
                
                // Render the forecast
                if (isDaily) {
                    this.renderDailyForecast(data.forecast, forecastType);
                } else {
                    this.renderHourlyForecast(data.forecast, forecastType);
                }
                
                // Add historical context
                this.addHistoricalContext();
            })
            .catch(error => {
                console.error('Error loading forecast data:', error);
                document.getElementById('forecast-container').innerHTML = 
                    `<div class="error">Error loading forecast data: ${error.message}</div>`;
            });
    },
    
    // Render daily forecast view
    renderDailyForecast: function(forecastData, forecastType) {
        // Process data for chart
        const dates = Object.keys(forecastData).sort();
        const calls = dates.map(date => forecastData[date].calls);
        const chats = dates.map(date => forecastData[date].chats);
        const agents = dates.map(date => forecastData[date].required_agents);
        
        // Format for display (convert YYYY-MM-DD to DD.MM)
        const formattedDates = dates.map(date => {
            const parts = date.split('-');
            return `${parts[2]}.${parts[1]}`;
        });
        
        // Create stacked chart for volume
        Highcharts.chart('forecast-volume-chart', {
            chart: {
                type: 'column'
            },
            title: {
                text: `${this.capitalizeFirstLetter(forecastType)} Forecast - Contact Volume`
            },
            xAxis: {
                categories: formattedDates,
                title: {
                    text: 'Date'
                }
            },
            yAxis: {
                min: 0,
                title: {
                    text: 'Number of Contacts'
                },
                stackLabels: {
                    enabled: true,
                    style: {
                        fontWeight: 'bold'
                    }
                }
            },
            tooltip: {
                headerFormat: '<b>{point.x}</b><br/>',
                pointFormat: '{series.name}: {point.y}<br/>Total: {point.stackTotal}'
            },
            plotOptions: {
                column: {
                    stacking: 'normal',
                    dataLabels: {
                        enabled: true
                    }
                }
            },
            series: [{
                name: 'Calls',
                data: calls,
                color: '#7CB5EC'
            }, {
                name: 'Chats',
                data: chats,
                color: '#90ED7D'
            }]
        });
        
        // Create line chart for required agents
        Highcharts.chart('forecast-agents-chart', {
            chart: {
                type: 'line'
            },
            title: {
                text: `${this.capitalizeFirstLetter(forecastType)} Forecast - Required Agents`
            },
            xAxis: {
                categories: formattedDates,
                title: {
                    text: 'Date'
                }
            },
            yAxis: {
                title: {
                    text: 'Number of Agents'
                }
            },
            series: [{
                name: 'Required Agents',
                data: agents,
                color: '#F45B5B',
                dataLabels: {
                    enabled: true
                }
            }]
        });
        
        // Create detailed table
        this.renderForecastTable(forecastData, dates);
    },
    
    // Render hourly forecast view
    renderHourlyForecast: function(forecastData, forecastType) {
        // Process data for heatmap
        const dates = Object.keys(forecastData).sort();
        const hours = Array.from({length: 24}, (_, i) => i);
        
        // Create data for calls heatmap
        const callsData = [];
        dates.forEach(date => {
            hours.forEach(hour => {
                callsData.push([hour, dates.indexOf(date), forecastData[date][hour]?.calls || 0]);
            });
        });
        
        // Format for display (convert YYYY-MM-DD to DD.MM)
        const formattedDates = dates.map(date => {
            const parts = date.split('-');
            return `${parts[2]}.${parts[1]}`;
        });
        
        // Create heatmap for calls
        Highcharts.chart('forecast-hourly-chart', {
            chart: {
                type: 'heatmap',
                marginTop: 40,
                marginBottom: 80
            },
            title: {
                text: `${this.capitalizeFirstLetter(forecastType)} Forecast - Hourly Call Volume`
            },
            xAxis: {
                categories: hours.map(h => `${h}:00`),
                title: {
                    text: 'Hour of Day'
                }
            },
            yAxis: {
                categories: formattedDates,
                title: {
                    text: 'Date'
                }
            },
            colorAxis: {
                min: 0,
                minColor: '#FFFFFF',
                maxColor: '#7CB5EC'
            },
            legend: {
                align: 'right',
                layout: 'vertical',
                margin: 0,
                verticalAlign: 'top',
                y: 25,
                symbolHeight: 280
            },
            tooltip: {
                formatter: function () {
                    return `<b>${this.series.yAxis.categories[this.point.y]}</b> at <b>${this.series.xAxis.categories[this.point.x]}</b><br/>
                            Calls: <b>${this.point.value}</b>`;
                }
            },
            series: [{
                name: 'Calls per Hour',
                borderWidth: 1,
                data: callsData,
                dataLabels: {
                    enabled: true,
                    color: '#000000'
                }
            }]
        });
    },
    
    // Render detailed forecast table
    renderForecastTable: function(forecastData, dates) {
        const tableContainer = document.getElementById('forecast-table-container');
        
        let tableHTML = `
            <div class="table-responsive mt-4">
                <table class="table table-bordered table-striped">
                    <thead class="thead-dark">
                        <tr>
                            <th>Date</th>
                            <th>Calls</th>
                            <th>Chats</th>
                            <th>Total</th>
                            <th>AHT (min)</th>
                            <th>SL (%)</th>
                            <th>FRT (min)</th>
                            <th>Abandon (%)</th>
                            <th>Required Agents</th>
                        </tr>
                    </thead>
                    <tbody>
        `;
        
        dates.forEach(date => {
            const forecast = forecastData[date];
            const formattedDate = date.split('-').reverse().join('.');
            const totalContacts = forecast.calls + forecast.chats;
            const abandonRate = forecast.calls > 0 ? 
                ((forecast.abandoned / forecast.calls) * 100).toFixed(1) : '0.0';
            
            tableHTML += `
                <tr>
                    <td>${formattedDate}</td>
                    <td>${forecast.calls}</td>
                    <td>${forecast.chats}</td>
                    <td>${totalContacts}</td>
                    <td>${(forecast.aht / 60).toFixed(1)}</td>
                    <td>${forecast.sl}%</td>
                    <td>${(forecast.frt / 60).toFixed(1)}</td>
                    <td>${abandonRate}%</td>
                    <td>${forecast.required_agents}</td>
                </tr>
            `;
        });
        
        tableHTML += `
                    </tbody>
                </table>
            </div>
        `;
        
        tableContainer.innerHTML = tableHTML;
    },
    
    // Add historical context for comparison
    addHistoricalContext: function() {
        if (!this.historicalData) return;
        
        const contextContainer = document.getElementById('historical-context');
        if (!contextContainer) return;
        
        let contextHTML = `
            <div class="card mt-4">
                <div class="card-header bg-info text-white">
                    Historical Context
                </div>
                <div class="card-body">
                    <div class="row">
                        <div class="col-md-6">
                            <h5>Call Statistics</h5>
                            <ul class="list-group">
                                <li class="list-group-item d-flex justify-content-between align-items-center">
                                    Average Daily Calls
                                    <span class="badge bg-primary rounded-pill">${Math.round(this.historicalData.calls.avg)}</span>
                                </li>
                                <li class="list-group-item d-flex justify-content-between align-items-center">
                                    Average AHT (min)
                                    <span class="badge bg-primary rounded-pill">${(this.historicalData.aht.avg / 60).toFixed(1)}</span>
                                </li>
                                <li class="list-group-item d-flex justify-content-between align-items-center">
                                    Average SL %
                                    <span class="badge bg-primary rounded-pill">${this.historicalData.sl.avg.toFixed(1)}%</span>
                                </li>
                            </ul>
                        </div>
                        <div class="col-md-6">
                            <h5>Chat Statistics</h5>
                            <ul class="list-group">
                                <li class="list-group-item d-flex justify-content-between align-items-center">
                                    Average Daily Chats
                                    <span class="badge bg-success rounded-pill">${Math.round(this.historicalData.chats.avg)}</span>
                                </li>
                            </ul>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        contextContainer.innerHTML = contextHTML;
    },
    
    // Helper to capitalize first letter
    capitalizeFirstLetter: function(string) {
        return string.charAt(0).toUpperCase() + string.slice(1);
    }
};

// Initialize the forecaster on document load
document.addEventListener('DOMContentLoaded', function() {
    Forecaster.init();
    
    // Load default forecast view if on forecast page
    if (document.getElementById('forecast-container')) {
        Forecaster.updateForecastView(FORECAST_TYPES.OPTIMAL);
    }
});

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = Forecaster;
}