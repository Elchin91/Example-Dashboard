// This file contains the JavaScript code that handles the dynamic aspects of the dashboard, such as fetching data from the server and updating the UI.

document.addEventListener("DOMContentLoaded", function() {
    const fetchData = (tab, startDate, endDate) => {
        fetch(`/daily/data/${tab}?start_date=${startDate}&end_date=${endDate}`)
            .then(response => response.json())
            .then(data => {
                updateDashboard(tab, data);
            })
            .catch(error => console.error('Error fetching data:', error));
    };

    const updateDashboard = (tab, data) => {
        const container = document.getElementById(`${tab}-data`);
        container.innerHTML = ''; // Clear previous data

        if (data.error) {
            container.innerHTML = `<p>${data.error}</p>`;
            return;
        }

        data.forEach(item => {
            const div = document.createElement('div');
            div.innerHTML = `${item.report_date || item.hour}: ${item.total_calls || item.avg_call_duration || item.total_chats || item.avg_chat_frt || item.sl || item.resolution_time_avg}`;
            container.appendChild(div);
        });
    };

    // Example usage
    const startDate = '2023-01-01';
    const endDate = '2023-01-31';
    const tabs = ['calls', 'aht', 'chats', 'frt', 'sl', 'rt'];

    tabs.forEach(tab => {
        fetchData(tab, startDate, endDate);
    });
});