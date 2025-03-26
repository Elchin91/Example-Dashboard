def execute_query(query, params, db_config):
    """Executes an SQL query with given parameters."""
    import mysql.connector
    try:
        connection = mysql.connector.connect(**db_config)
        cursor = connection.cursor(dictionary=True)
        cursor.execute(query, params)
        result = cursor.fetchall()
        return result
    finally:
        connection.close()