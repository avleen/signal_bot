#!/usr/bin/env python3

"""A collection of helper functions for interacting with Signal."""

import base64
import json
import os
import requests

def get_message_root(message):
    # Get the message body
    # Sometimes syncMessage is empty, so we need to check for that.
    # The two types of message body look like this:
    # {"envelope":
    #   {"source":"+<int>","sourceNumber":"+<int>","sourceUuid":"<uuid>","sourceName":"<string>","sourceDevice":3,"timestamp":<nanoseconds>,"syncMessage":{
    #       "sentMessage":{
    #           "destination":null,"destinationNumber":null,"destinationUuid":null,"timestamp":<nanoseconds>,"message":"<message>","expiresInSeconds":604800,"viewOnce":false,"groupInfo":{
    #               "groupId":"<string>","type":"DELIVER"}
    #           }
    #       }
    #   },"account":"+<int>"
    # }
    # {"envelope":{
    #   "source":"<uuid>","sourceNumber":null,"sourceUuid":"<uuid>","sourceName":"<string>","sourceDevice":1,"timestamp":<nanoseconds>,"dataMessage":{
    #       "timestamp":<nanoseconds>,"message":"<string>","expiresInSeconds":604800,"viewOnce":false,"groupInfo":{
    #           "groupId":"<string>","type":"DELIVER"}
    #       }
    #   },"account":"+<int>"
    # }
    msg_struct = None
    parsed = json.loads(message)['envelope']
    if 'dataMessage' in parsed:
        msg_struct = parsed['dataMessage']
    elif 'syncMessage' in parsed:
        try:
            msg_struct = parsed['syncMessage']['sentMessage']
        except KeyError:
            return None, None
    return msg_struct, parsed

def send_message(url, message, number, recipients, attachment=None):
    """Send a message to a Signal recipient or group.
    
    Args:
        message (str): The message to send.
        number (str): The recipient's phone number in E.123 international notation (+1234567890).
        recipients (list): A list of recipients to send the message to. This can be a single recipient
            or a list of recipients. Each recipient should be a phone number or in the format "group.<groupId>",
            where <groupId> is the base64-encoded group ID.
        attachment (str): The path to an attachment to send."""
    data = {'message': message, 'number': number, 'recipients': recipients}
    if attachment:
        # Check if the file exists
        if not os.path.exists(attachment):
            raise ValueError(f"Attachment {attachment} does not exist")
        # Convert the file to base64 as a utf-8 string
        with open(attachment, 'rb') as f:
            file = base64.b64encode(f.read()).decode('utf-8')
            data["base64_attachments"] = [f"{file}"]
    res = requests.post(f'http://{url}/v2/send', json=data,)
    return res

def convert_groupid(groupId):
    groupId = base64.b64encode(groupId.encode()).decode()
    return f"group.{groupId}"