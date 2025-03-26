# Flask Dashboard

This project is a Flask web application that visualizes SQL query results on a dashboard. It provides an interactive interface for users to view and analyze data from a MySQL database.

## Project Structure

```
flask-dashboard
├── app
│   ├── __init__.py          # Initializes the Flask application and sets up configuration and routes.
│   ├── routes.py            # Contains route definitions and logic for handling requests.
│   ├── static
│   │   ├── css
│   │   │   └── styles.css   # CSS styles for the dashboard.
│   │   └── js
│   │       └── main.js      # JavaScript code for dynamic aspects of the dashboard.
│   ├── templates
│   │   └── dashboard.html    # HTML template for the dashboard.
│   └── utils.py             # Utility functions for database interaction.
├── config.py                # Configuration settings for the application.
├── requirements.txt         # Python dependencies required for the project.
└── README.md                # Documentation for the project.
```

## Installation

1. Clone the repository:
   ```
   git clone <repository-url>
   cd flask-dashboard
   ```

2. Create a virtual environment:
   ```
   python -m venv venv
   ```

3. Activate the virtual environment:
   - On Windows:
     ```
     venv\Scripts\activate
     ```
   - On macOS/Linux:
     ```
     source venv/bin/activate
     ```

4. Install the required packages:
   ```
   pip install -r requirements.txt
   ```

## Configuration

Update the `config.py` file with your database connection details.

## Running the Application

To run the application, execute the following command:
```
python -m app
```

Visit `http://127.0.0.1:5000` in your web browser to access the dashboard.

## Usage

The dashboard allows users to visualize data from the database through various metrics. Users can select different tabs to view daily and hourly data.

## License

This project is licensed under the MIT License.