import base64
import requests


def upload():
    with open("../test_data/log_3124689.log", 'r') as f:
        data = f.read()
    url = "http://localhost:6006/api/log"
    pv = {
        'server_name': 'localhost-1',
        'map_name': 'pl_meme',
        'body': base64.b64encode(data.encode('utf-8')).decode('ascii'),
        'type': "gbans_log"
    }
    requests.post(url, json=pv, headers={
        "Authorization": "xxxxxxxxxxxxxxxxxxxx"
    })


if __name__ == "__main__":
    upload()
