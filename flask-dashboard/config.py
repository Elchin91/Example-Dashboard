from dotenv import load_dotenv
import os

load_dotenv()

class Config:
    DEBUG = os.getenv("DEBUG", "True") == "True"
    HOST = os.getenv("DB_HOST", "192.168.46.4")
    PORT = int(os.getenv("DB_PORT", 3306))
    USER = os.getenv("DB_USER", "pashapay")
    PASSWORD = os.getenv("DB_PASSWORD", "Q1w2e3r4!@#")
    DATABASE = os.getenv("DB_DATABASE", "report")