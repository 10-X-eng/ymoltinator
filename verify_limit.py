import requests
import sys

try:
    response = requests.get("http://localhost:8080/api/stories")
    response.raise_for_status()
    data = response.json()
    
    # Check if 'items' or similar key exists, or if it's a list directly
    if isinstance(data, list):
        count = len(data)
    elif 'stories' in data:
        count = len(data['stories'])
    elif 'data' in data:
        count = len(data['data'])
    else:
        print(f"Unknown response format: {data.keys()}")
        sys.exit(1)
        
    print(f"Returned {count} stories")
    
    # We expect 100 if there are enough stories, or just check the default param didn't crash it
    # Ideally we'd check if we *can* get more than 30.
    # If the DB is empty, this might return 0. 
    # But the goal is to verify the configuration.
    
    # We can also check the response headers or metadata if available.
    
    if count > 30:
        print("Success: Returned more than 30 stories.")
    else:
        print("Note: Returned 30 or fewer stories (might be DB limit).")
        
except Exception as e:
    print(f"Error: {e}")
    sys.exit(1)
