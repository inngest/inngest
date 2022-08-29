import json
import os


def run(event, steps, context) -> dict:
    return {
        "status": 200,
        "body": {
            "simple": os.environ.get('SIMPLE'),
            "quoted": os.environ.get('QUOTED'),
            "quotedEscapes": os.environ.get('QUOTED_ESCAPES'),
            "certificate": os.environ.get('CERTIFICATE'),
            "json":json.loads(os.environ.get('JSON'))
        }
    }
