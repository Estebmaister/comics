import os
import ssl

import pymysql
from sqlalchemy import create_engine, text
from sqlalchemy.engine.url import URL

# Absolute path to the SSL certificate
ssl_ca_path = './DigiCertGlobalRootG2.crt.pem'

DB_DRIVER = 'mysql+pymysql'
DB_USER = 'esteb'
DB_PASS = 'P4ssw0rd'
DB_HOST = 'comic-db.mysql.database.azure.com'
DB_PORT = 3306
DB_NAME = 'comics'

# Configure PyMySQL to use SSL
pymysql.install_as_MySQLdb()
pymysql.connections.SSL = {
    'ca': ssl_ca_path,
    'check_hostname': True,
}

# Construct connection URL
connection_url = URL.create(
    drivername=DB_DRIVER,
    username=DB_USER,
    password=DB_PASS,
    host=DB_HOST,
    port=DB_PORT,
    database=DB_NAME,
)

# Create engine with SSL configuration
engine = create_engine(
    connection_url,
    connect_args={
        'ssl': {
            'ca': ssl_ca_path,
            'check_hostname': True,
        }
    }
)

# Detailed connection test with comprehensive error handling


def test_database_connection():
    try:
        print("Attempting to connect to the database...")
        with engine.connect() as connection:
            # Use text() to create an executable SQL statement
            result = connection.execute(text("SELECT 1"))
            print("Connection successful!")
            print("Query result:", result.fetchone())

    except pymysql.err.OperationalError as oe:
        print("Operational Error:")
        print(f"Error Code: {oe.args[0]}")
        print(f"Error Message: {oe.args[1]}")
        print("\nPossible causes:")
        print("1. Incorrect SSL certificate")
        print("2. Network connectivity issues")
        print("3. Firewall restrictions")

    except Exception as e:
        print("Unexpected error occurred:")
        print(f"Error Type: {type(e)}")
        print(f"Error Details: {str(e)}")
        import traceback
        traceback.print_exc()


# Run the connection test
if __name__ == "__main__":
    test_database_connection()
