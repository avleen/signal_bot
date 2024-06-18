import unittest
from unittest import mock
from unittest.mock import patch

# Set some environment variables to ensure the tests run correctly
import os
os.environ['IMAGEDIR'] = 'test_images'
os.environ['STATEDB'] = 'test_messages.db'
os.environ['HOURS'] = '5'
os.environ['PHONE'] = '1234567890'
os.environ['URL'] = 'localhost:8080'
os.environ['REST_URL'] = 'localhost:8080'
import helpers
import json
import sqlite3
import wsclient

TEST_LOG_LINES = [
                '{"envelope": {"timestamp": 1629878400000, "sourceName": "Author1", "sourceNumber": "+1234567890", "groupInfo": {"groupId": "abc123"}, "dataMessage": {"message": "Message1"}}}',
                '{"envelope": {"timestamp": 1629879000000, "sourceName": "Author2", "sourceNumber": "+1234567890", "groupInfo": {"groupId": "abc123"}, "dataMessage": {"message": "Message2"}}}',
                '{"envelope": {"timestamp": 1629879600000, "sourceName": "Author3", "sourceNumber": "+1234567890", "groupInfo": {"groupId": "abc123"}, "dataMessage": {"message": "Message3"}}}'
            ]

class TestWSClient(unittest.TestCase):

    @patch('wsclient.cleanup_statefile')
    @patch('wsclient.summarize.summarize')
    @patch('wsclient.summarize.get_log_lines')
    @patch('wsclient.send_message')
    def test_on_message_summary_request(self, mock_send_message, mock_get_log_lines, mock_summarize, mock_cleanup_statefile):
        # Test when a summary request is received
        ws = None
        message = '{"envelope": {"timestamp": 1629879600000, "sourceName": "Example Name", "sourceNumber": "+123456789", "dataMessage": {"message": "!summary 5h", "groupInfo": {"groupId": "abc123"}}}}'
        b64_groupId = helpers.convert_groupid("abc123")
        mock_get_log_lines.return_value = "Summary"
        mock_summarize.return_value = "Summary"
        mock_cleanup_statefile.return_value = None
        wsclient.on_message(ws, message)
        mock_send_message.assert_called_once_with(os.environ['URL'], "Summary", os.environ["PHONE"], [b64_groupId])

    @patch('wsclient.cleanup_statefile')
    @patch('wsclient.send_message')
    def test_on_message_non_summary_request(self, mock_send_message, mock_cleanup_statefile):
        # Test when a non-summary request is received
        ws = None
        message = '{"envelope": {"timestamp": 1629879600000, "sourceName": "Example Name", "sourceNumber": "+123456789", "dataMessage": {"message": "Hello", "groupInfo": {"groupId": "abc123"}}}}'
        mock_cleanup_statefile.return_value = None
        wsclient.on_message(ws, message)
        mock_send_message.assert_not_called()

    @patch('wsclient.cleanup_statefile')
    @patch('wsclient.send_message')
    def test_on_message_no_group_info(self, mock_send_message, mock_cleanup_statefile):
        # Test when the message does not have groupInfo
        ws = None
        message = '{"envelope": {"timestamp": 1629879600000, "sourceName": "Example Name", "sourceNumber": "+123456789", "dataMessage": {"message": "!summary 5h"}}}'
        mock_cleanup_statefile.return_value = None
        wsclient.on_message(ws, message)
        mock_send_message.assert_not_called()

    @patch('wsclient.cleanup_statefile')
    @patch('wsclient.send_message')
    def test_on_message_empty_message_root(self, mock_send_message, mock_cleanup_statefile):
        # Test when the message root is empty
        ws = None
        message = '{"envelope": {"timestamp": 1629879600000, "sourceName": "Example Name", "sourceNumber": "+123456789"}}'
        mock_cleanup_statefile.return_value = None
        wsclient.on_message(ws, message)
        mock_send_message.assert_not_called()

    @patch('wsclient.cleanup_statefile')
    @patch('summarize.get_log_lines')
    @patch('wsclient.send_message')
    def test_on_message_error(self, mock_send_message, mock_get_log_lines, mock_cleanup_statefile):
        # Test when an error occurs during summary generation
        ws = None
        message = '{"envelope": {"timestamp": 1629879600000, "sourceName": "Example Name", "sourceNumber": "+123456789", "dataMessage": {"message": "!summary 5h", "groupInfo": {"groupId": "abc123"}}}}'
        b64_groupId = helpers.convert_groupid("abc123")
        mock_get_log_lines.side_effect = ValueError("Summary failed")
        mock_cleanup_statefile.return_value = None

        wsclient.on_message(ws, message)
        mock_send_message.assert_called_once_with(os.environ['URL'], "ERROR: Summary failed", os.environ["PHONE"], [b64_groupId])

class TestGetCountHours(unittest.TestCase):

    def test_no_count_or_hours(self):
        # Test when the message does not contain count or hours
        msg = "!summary"
        count, hours, question = wsclient.get_count_hours(msg)
        self.assertIsNone(count)
        self.assertEqual(hours, 24)

    def test_count_only(self):
        # Test when the message contains count but not hours
        msg = "!summary 5"
        count, hours, question = wsclient.get_count_hours(msg)
        self.assertEqual(count, 5)
        self.assertEqual(hours, 168)

    def test_hours_only(self):
        # Test when the message contains hours but not count
        msg = "!summary 5h"
        count, hours, question = wsclient.get_count_hours(msg)
        self.assertIsNone(count)
        self.assertEqual(hours, 5)

    def test_valid_question(self):
        # Test when the message contains a valid question
        msg = "!summary How are you?"
        count, hours, question = wsclient.get_count_hours(msg)
        self.assertIsNone(count)
        self.assertEqual(hours, 168)
        self.assertEqual(question, "How are you?")

    def test_invalid_question(self):
        # Test when the message contains an invalid question
        msg = "!summary How are you"
        with self.assertRaises(ValueError):
            wsclient.get_count_hours(msg)

class TestPersistMessage(unittest.TestCase):
    def __init__(self, *args, **kwargs):
        super(TestPersistMessage, self).__init__(*args, **kwargs)
        # Delete and initialize the test_messages.db file
        if os.path.exists('test_messages.db'):
            os.remove('test_messages.db')
        conn = sqlite3.connect('test_messages.db')
        c = conn.cursor()
        c.execute("""CREATE TABLE `messages` (
  `id` integer not null primary key autoincrement,
  `timestamp` UNSIGNED BIG INT null,
  `sourceNumber` TEXT null,
  `sourceName` TEXT not null,
  `message` TEXT not null,
  `groupId` TEXT not null,
  `created_at` datetime not null default CURRENT_TIMESTAMP)""")
        conn.commit()
        for line in TEST_LOG_LINES:
            line = json.loads(line)
            c.execute("INSERT INTO messages (timestamp, sourceNumber, sourceName, message, groupId) VALUES (?, ?, ?, ?, ?)",
                (line['envelope']['timestamp'],
                line['envelope']['sourceNumber'],
                line['envelope']['sourceName'],
                line['envelope']['dataMessage']['message'],
                line['envelope']['groupInfo']['groupId'],))
            conn.commit()
        conn.close()

    @patch('wsclient.sqlite3.connect')
    def test_persist_message(self, mock_connect):
        # Test persisting a message to the database
        msg_root = {
            "message": "Hello",
            "groupInfo": {
                "groupId": "abc123"
            }
        }
        message = '{"envelope": {"timestamp": 1629879600000, "sourceName": "Example Name", "sourceNumber": "+123456789"}, "dataMessage": {"message": "Hello", "groupInfo": {"groupId": "abc123"}}}'
        wsclient.persist_message(msg_root, message)
        mock_connect.assert_called_once_with(wsclient.statedb)
        mock_connect.return_value.cursor.assert_called_once()
        mock_connect.return_value.cursor.return_value.execute.assert_called_once_with(
            "INSERT INTO messages (timestamp, sourceNumber, sourceName, message, groupId) VALUES (?, ?, ?, ?, ?)",
            (1629879600000, "+123456789", "Example Name", "Hello", "abc123")
        )
        mock_connect.return_value.commit.assert_called_once()
        mock_connect.return_value.close.assert_called_once()

class TestCleanupStatefile(unittest.TestCase):

    @mock.patch('time.time', mock.MagicMock(return_value=(1629274800 + 604800)))
    @patch('wsclient.sqlite3.connect')
    def test_cleanup_statefile(self, mock_connect):
        # Test cleaning up the state file
        wsclient.max_age = 168
        wsclient.statedb = 'test_messages.db'
        wsclient.cleanup_statefile()
        mock_connect.assert_called_once_with(wsclient.statedb)
        mock_connect.return_value.cursor.return_value.execute.assert_called_once_with("DELETE FROM messages WHERE timestamp < ?", (1629274800000,))
        mock_connect.return_value.commit.assert_called_once()
        mock_connect.return_value.close.assert_called_once()

if __name__ == '__main__':
    unittest.main()