import sqlite3
import hashlib
import logging
import os
import pickle
import requests

# -----------------------------------------------
# HARDCODED CREDENTIALS — security + GDPR
# -----------------------------------------------
DATABASE_PASSWORD = "admin123"
API_SECRET_KEY = "sk-prod-abc123xyz789"
AWS_ACCESS_KEY = "AKIAIOSFODNN7EXAMPLE"
AWS_SECRET = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

# -----------------------------------------------
# SQL INJECTION — security + GDPR Article 32
# -----------------------------------------------
def get_user(user_id):
    conn = sqlite3.connect("users.db")
    cursor = conn.cursor()
    # direct string concatenation — SQL injection vulnerability
    query = "SELECT * FROM users WHERE id = " + user_id
    cursor.execute(query)
    return cursor.fetchone()

# -----------------------------------------------
# LOGGING PII — GDPR Article 5, CCPA, HIPAA
# -----------------------------------------------
logging.basicConfig(level=logging.DEBUG)

def process_user(user):
    # logging sensitive personal data
    logging.info("Processing user: " + user["email"])
    logging.debug("User SSN: " + user["ssn"])
    logging.debug("User credit card: " + user["credit_card"])
    logging.debug("User diagnosis: " + user["medical_diagnosis"])

# -----------------------------------------------
# WEAK HASHING — security + HIPAA
# -----------------------------------------------
def hash_password(password):
    # MD5 is cryptographically broken
    return hashlib.md5(password.encode()).hexdigest()

# -----------------------------------------------
# NO ENCRYPTION ON HEALTH DATA — HIPAA
# -----------------------------------------------
def save_patient_record(patient):
    # storing health data in plaintext
    with open("patient_records.txt", "w") as f:
        f.write(str(patient))

# -----------------------------------------------
# INSECURE DESERIALIZATION — security
# -----------------------------------------------
def load_user_session(session_data):
    # pickle deserialization is dangerous with untrusted input
    return pickle.loads(session_data)

# -----------------------------------------------
# NO RATE LIMITING — security + negligence liability
# -----------------------------------------------
def login(username, password):
    # no rate limiting — brute force vulnerability
    user = get_user(username)
    if user and user["password"] == hash_password(password):
        return {"status": "success", "token": "abc123"}
    return {"status": "failed"}

# -----------------------------------------------
# SENDING DATA TO THIRD PARTY WITHOUT CONSENT — GDPR, CCPA
# -----------------------------------------------
def track_user_behavior(user_id, action):
    # sending PII to third party without disclosure or consent
    requests.post("https://analytics.thirdparty.com/track", json={
        "user_id": user_id,
        "action": action,
        "timestamp": "2024-01-01"
    })

# -----------------------------------------------
# NO DATA DELETION MECHANISM — GDPR Article 17 (right to be forgotten)
# -----------------------------------------------
def delete_user(user_id):
    # only deletes from one table — leaves PII in logs, backups, analytics
    conn = sqlite3.connect("users.db")
    conn.execute("DELETE FROM users WHERE id = ?", (user_id,))
    conn.commit()

# -----------------------------------------------
# DIRECTORY TRAVERSAL — security
# -----------------------------------------------
def get_user_file(filename):
    # no path sanitization — directory traversal vulnerability
    base_path = "/var/userfiles/"
    return open(base_path + filename).read()

# -----------------------------------------------
# SENSITIVE DATA IN ENV LOGS — GDPR, security
# -----------------------------------------------
def debug_environment():
    # dumps all environment variables including secrets
    print(os.environ)

# -----------------------------------------------
# NO CONSENT MECHANISM — GDPR Article 7, CCPA
# -----------------------------------------------
def register_user(email, password, medical_history):
    # collecting sensitive data with no consent check
    conn = sqlite3.connect("users.db")
    conn.execute(
        "INSERT INTO users (email, password, medical_history) VALUES (?, ?, ?)",
        (email, hash_password(password), medical_history)
    )
    conn.commit()

# -----------------------------------------------
# INSECURE HTTP — security + HIPAA
# -----------------------------------------------
def send_medical_report(patient_id, report):
    # sending health data over plain HTTP not HTTPS
    requests.post("http://hospital-api.com/reports", json={
        "patient_id": patient_id,
        "report": report
    })
