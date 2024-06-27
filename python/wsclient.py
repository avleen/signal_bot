#!/usr/bin/env python3

import argparse
import imagine_openai
import json
import os
import rel
import requests
import sqlite3
import summarize
import time
import traceback
import websocket

from helpers import convert_groupid, get_message_root, send_message

# Get the url, hours to summarise and phone number from the environment
imagedir = os.environ['IMAGEDIR']
statedb = os.environ['STATEDB']
hours = os.environ['HOURS']
phone = os.environ['PHONE']
url = os.environ['URL']
rest_url = os.environ['REST_URL']
max_age = 168 # 1 week

def get_count_hours(msg):
    """If the message contains a count or hours, return them. Otherwise, return default values."""
    count = None
    hours = 24
    question = None
    msg = msg.split()
    if len(msg) > 1:
        # We got a more complex request. It's either the number of hours,
        # or the number of messages, or a question. Let's try to figure out what it is.

        # First, if we have a number, it's the number of messages.
        if len(msg) == 2 and msg[1].isdigit():
            count = int(msg[1])
            hours = 168 # Default to 1 week if count is set
        # Next if we have a number followed by an h, it's the number of hours.
        elif len(msg) == 2 and msg[1][-1] == 'h' and msg[1][:-1].isdigit():
            hours = int(msg[1][:-1])
        # Finally if the text ends in a question mark, it's a question.
        elif msg[-1].endswith('?'):
            hours = 168 # Default to 1 week if question is set
            question = ' '.join(msg[1:])
        else:
            raise ValueError("Was that a question?")
    return count, hours, question

def cleanup_statefile():
    # Delete entries from the database that are older than max_age hours
    # Get the current time in ns
    now = int(time.time() * 1000)
    max_age_ts = now - (max_age * 3600 * 1000)
    conn = sqlite3.connect(statedb)
    c = conn.cursor()
    c.execute("DELETE FROM messages WHERE timestamp < ?", (max_age_ts,))
    conn.commit()
    conn.close()

def persist_message(msg_root, message):
    # Parse the message and save it to the database
    parsed = json.loads(message)['envelope']
    conn = sqlite3.connect(statedb)
    c = conn.cursor()
    c.execute("INSERT INTO messages (timestamp, sourceNumber, sourceName, message, groupId) VALUES (?, ?, ?, ?, ?)",
                (parsed['timestamp'],
                 parsed['sourceNumber'],
                 parsed['sourceName'],
                 msg_root['message'],
                 msg_root['groupInfo']['groupId'],))
    conn.commit()
    conn.close()

def on_message(ws, message):
    print(f"INFO: received message: {str(message)}")
    groupId = None

    # Get the message root. If it is empty, return.
    msg_root, parsed = get_message_root(message)
    if not msg_root:
        return

    # Skip messages with no groupInfo, they are not group messages.
    if 'groupInfo' not in msg_root:
        return

    # Convert the groupId to base64
    groupId = convert_groupid(msg_root['groupInfo']['groupId'])

    # groupId should always be set here. But if it is not, return.
    if groupId is None:
        return

    # Check if the message is a summary request.
    # If so, get the groupId and the hours or count to summarize.
    msg = msg_root['message']
    # First clean up the state file, if this was a real message
    if msg:
        try:
            persist_message(msg_root, message)
        except Exception as e:
            print(f"ERROR: {str(e)} {traceback.print_exc()}")
            return
        cleanup_statefile()

    # Now see if this is a summary request
    if msg and msg.startswith('!summary'):
        # Get the count or hours from the message
        count, hours, question = get_count_hours(msg)
        try:
            print(f"INFO: Summary requested by {parsed['sourceName']}, count={count}, hours={hours}, question={question}")
            summary = summarize.get_summary(count, hours, question, statedb)
            # Send the summary to the server using a POST request, with body type application/json
            res = send_message(url, summary, phone, [groupId])
            if res.status_code != 201:
                print(f"ERROR: {res.text}")
            return
        except Exception as e:
            print(f"ERROR: {str(e)}")
            res = send_message(url, f"ERROR: {str(e)}", phone, [groupId])
    elif msg and msg.startswith('!imagine'):
        try:
            prompt = msg.split('!imagine ')[1]
            print(f"INFO: Image requested by {parsed['sourceName']}: {prompt}")
            image_file, revised_prompt = imagine_openai.imagine(prompt, imagedir, parsed['sourceName'])
        except Exception as e:
            print(f"ERROR: {str(e)} {traceback.print_exc()}")
            res = send_message(url, f"ERROR: {str(e)}", phone, [groupId])
            return
        res = send_message(url, f"Image generated with revised prompt: '{revised_prompt}'", phone, [groupId], image_file)
        if res.status_code != 201:
            print(f"ERROR: {res.text}")
        
def on_error(ws, error):
    print('ERROR:'+ str(error))

def on_close(ws, close_status_code, close_reason):
    print('INFO: closed connection')

def on_open(ws):
    print('INFO: opened connection')

def fetch_rest(url):
    # Fetch the responses from the REST API, save them and exit.
    res = requests.get(f'http://{rest_url}/v1/receive/{phone}')
    
    # res.text is a string representing a list of dictionaries.
    # Iterate over the list and save each dictionary to the statefile.
    for message in res.json():
        msg_root, parsed = get_message_root(message)
        print(f"INFO: received message: {message}")
        persist_message(msg_root, message)

def main(mode='websocket'):
    print('INFO: Starting')
    if mode == 'rest':
        print('INFO: Using REST API')
        fetch_rest(rest_url)
        return
    
    print('INFO: Using websocket')
    # Connect to the signal-api websocket and start collecting messages.
    # Print each line to both the screen and save it to the state file.
    # websocket.enableTrace(True)
    ws = websocket.WebSocketApp(f'ws://{url}/v1/receive/{phone}',
                                on_message=on_message,
                                on_open=on_open,
                                on_error=on_error,
                                on_close=on_close)
    
    ws.run_forever(dispatcher=rel, reconnect=1)
    rel.set_sleep(0.1)
    rel.set_turbo(0.1)
    rel.signal(2, rel.abort)
    rel.dispatch()

if __name__ == "__main__":
    # Command line arguments
    # --mode: websocket or rest
    parser = argparse.ArgumentParser(description='Signal API client')
    parser.add_argument('--mode', type=str, default='websocket', help='websocket or rest')
    args = parser.parse_args()
    main(args.mode)