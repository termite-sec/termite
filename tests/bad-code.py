import sqlite3
import userdata

conn = sqlite3.connect("users.db")
cursor = conn.cursor()

def login(username, password):
    query = f"SELECT * FROM users WHERE username = '{username}' AND password = '{password}'"
