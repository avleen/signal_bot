#!/usr/bin/env python3

import argparse
import os
import sqlite3
import time
import traceback
import vertexai
from vertexai.generative_models import GenerativeModel
from google.cloud.aiplatform_v1beta1.types import SafetySetting, HarmCategory

project_id = os.environ["PROJECT_ID"]
location = os.environ["LOCATION"]

def fetch_from_db(statedb, count, starttime):
    # Open the state database and select lines from it.
    # First, if hours is set, get the lines since $starttime.
    # If count is set, get the last $count lines.
    conn = sqlite3.connect(statedb)
    c = conn.cursor()
    if count is not None:
        res = c.execute("SELECT sourceName || ': ' || message FROM messages ORDER BY timestamp DESC LIMIT ?", (count,))
        messages = res.fetchall()
        # Reverse the messages so they are in the correct order
        messages.reverse()
    elif starttime is not None:
        res = c.execute("SELECT sourceName || ': ' || message FROM messages WHERE timestamp >= ? ORDER BY timestamp ASC", (starttime,))
        messages = res.fetchall()
    else:
        raise ValueError("Either hours or count must be provided")
    conn.close()
    return messages

def get_log_lines(statedb, hours=24, count=None, currenttime=time.time()*1000):
    """Get the log lines from the statefile that are within the last $hours hours.
    We set a default for current time to be the current time to make testing easier."""

    # If both hours and count are None, return an error.
    if hours is None and count is None:
        raise ValueError("Either hours or count must be provided")
    
    # If hours is longer than 1 week, return an error.
    if hours is not None and hours > 168:
        raise ValueError("Hours cannot be more than 168")
    
    # Get the time that is $hours ago
    # If hours is none, set it to the maximum of 1 week
    if hours is None:
        hours = 168
    if hours == 0:
        starttime = 0
    else:
        starttime = currenttime - (3600*hours*1000)

    # Get the messages from the database
    messages = fetch_from_db(statedb, count, starttime)

    # Convert messages to text. It's a list of tuples, and each tuple has one element: the message.
    text = "\n".join([msg[0] for msg in messages])
    # Return the text
    return text

def get_summary(count, hours, question, statedb):
    try:
        text = get_log_lines(statedb, hours, count)
    except ValueError as e:
        # Send the error to Signal and return
        print(f"ERROR: {str(e)} {traceback.print_exc()}")
        raise e

    summary = summarize(text, question)
    return summary

def summarize(text, question=None):
    vertexai.init(project=project_id, location=location)
    if question:
        prompt = f"""Answer the following question, based on the text that follows it:
{question}

{text}"""
    else:
        # If a file called prompt_prefix.txt exists, us it as the prompt prefix
        prompt_prefix = ""
        if os.path.exists("prompt_prefix.txt"):
            with open("prompt_prefix.txt", "r") as f:
                prompt_prefix = f.read()
        prompt = f"""{prompt_prefix}

{text}
"""
    generative_model = GenerativeModel("gemini-1.5-flash-001")
    safety_settings = {
        harm_category: SafetySetting.HarmBlockThreshold(SafetySetting.HarmBlockThreshold.BLOCK_ONLY_HIGH)
        for harm_category in iter(HarmCategory)
    }
    response = generative_model.generate_content(prompt, safety_settings=safety_settings)
    print(f"DEBUG: summary response: {response}")
    if response.candidates[0].finish_reason.name == "SAFETY":
        stop_reason = next((d for d in response.candidates[0].safety_ratings if 'blocked' in d))
        return f"ERROR: The response was flagged as unsafe: {stop_reason}"

    return response.candidates[0].content.parts[0].text

if __name__ == "__main__":
    # Get the statefile name and number of hours to summarize from command line
    parser = argparse.ArgumentParser(description="Summarize messages")
    parser.add_argument("--statedb", help="State DB file name", required=True)
    parser.add_argument("--hours", type=int, help="Number of hours to scan", default=24)
    parser.add_argument("--question", type=str, help="Question to ask", default=None)
    args = parser.parse_args()

    text = get_log_lines(args.statedb, args.hours)
    print(summarize(text, args.question))
