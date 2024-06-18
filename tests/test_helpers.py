import unittest
from unittest import mock
from unittest.mock import patch

import os
os.environ['PHONE'] = '1234567890'
import base64
import helpers

class TestGetMessageRoot(unittest.TestCase):

    def test_data_message(self):
        # Test when the message is a data message
        message = '{"envelope": {"timestamp": 1629879600000, "sourceName": "Example Name", "sourceNumber": "+123456789", "dataMessage": {"message": "Hello", "groupInfo": {"groupId": "abc123"}}}}'
        msg_root, parsed = helpers.get_message_root(message)
        self.assertEqual(msg_root, {"message": "Hello", "groupInfo": {"groupId": "abc123"}})
        self.assertEqual(parsed['sourceName'], "Example Name")

    def test_sync_message(self):
        # Test when the message is a sync message
        message = '{"envelope": {"timestamp": 1629879600000, "sourceName": "Example Name", "sourceNumber": "+123456789", "syncMessage": {"sentMessage": {"message": "Hello", "groupInfo": {"groupId": "abc123"}}}}}'
        msg_root, parsed = helpers.get_message_root(message)
        self.assertEqual(msg_root, {"message": "Hello", "groupInfo": {"groupId": "abc123"}})
        self.assertEqual(parsed['sourceName'], "Example Name")

    def test_empty_message(self):
        # Test when the message is empty
        message = '{"envelope": {"timestamp": 1629879600000, "sourceName": "Example Name"}}'
        msg_root, parsed = helpers.get_message_root(message)
        self.assertIsNone(msg_root)
        self.assertEqual(parsed['sourceName'], "Example Name")

    def test_invalid_message(self):
        # Test when the message is invalid
        message = '{"envelope": {"timestamp": 1629879600000, "sourceName": "Example Name", "sourceNumber": "+123456789", "invalidMessage": {"message": "Hello", "groupInfo": {"groupId": "abc123"}}}}'
        msg_root, parsed = helpers.get_message_root(message)
        self.assertIsNone(msg_root)
        self.assertEqual(parsed['sourceName'], "Example Name")

class TestSendMessage(unittest.TestCase):

    @patch('requests.post')
    def test_send_message_without_attachment(self, mock_post):
        # Test sending a message without an attachment
        url = "example.com"
        message = "Hello, World!"
        number = "+1234567890"
        recipients = ["+9876543210"]
        expected_data = {'message': message, 'number': number, 'recipients': recipients}
        
        mock_post.return_value.status_code = 200
        res = helpers.send_message(url, message, number, recipients)
        
        mock_post.assert_called_once_with(f'http://{url}/v2/send', json=expected_data)
        self.assertEqual(res.status_code, 200)

    @patch('requests.post')
    @patch('base64.decode')
    def test_send_message_with_attachment(self, mock_b64encode, mock_post):
        # Test sending a message with an attachment
        url = "example.com"
        message = "Hello, World!"
        number = "+1234567890"
        recipients = ["+9876543210"]
        attachment = "Dockerfile"
        b64_data = b'Test string'
        b64_encoded = base64.b64encode(b64_data).decode('utf-8')
        expected_data = {'message': message, 'number': number, 'recipients': recipients, 'base64_attachments': [b64_encoded]}
        
        with mock.patch('builtins.open', mock.mock_open(read_data=b64_data)) as mock_open:
            mock_post.return_value.status_code = 200
            # Mock opening attachment and converting to base64
            mock_b64encode.return_value = b64_data

            res = helpers.send_message(url, message, number, recipients, attachment)
            
            mock_post.assert_called_once_with(f'http://{url}/v2/send', json=expected_data)
            mock_open.assert_called_once_with(attachment, 'rb')
            self.assertEqual(res.status_code, 200)

    def test_send_message_with_invalid_attachment(self):
        # Test sending a message with an invalid attachment
        url = "example.com"
        message = "Hello, World!"
        number = "+1234567890"
        recipients = ["+9876543210"]
        attachment = "/path/to/nonexistent_attachment.txt"
        
        with self.assertRaises(ValueError) as cm:
            helpers.send_message(url, message, number, recipients, attachment)
        
        self.assertEqual(str(cm.exception), f"Attachment {attachment} does not exist")