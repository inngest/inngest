import sys
import json

from app import run

def main():
    args = sys.argv.pop()
    json_payload = json.loads(args)

    try:
        response = run(json_payload["event"], json_payload["steps"], json_payload["ctx"])
    except Exception as e:
        response = {
            "status": 500,
            "body": str(e)
        }

    if isinstance(response,str):
        print(json.dumps({"status": 200, "body": response}))
    else:
        print(json.dumps(response))

main()