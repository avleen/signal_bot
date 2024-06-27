import os
os.environ['IMAGEDIR'] = 'test_images'
os.environ['STATEDB'] = 'test_messages.db'
os.environ['HOURS'] = '5'
os.environ['PHONE'] = '1234567890'
os.environ['URL'] = 'localhost:8080'
os.environ['REST_URL'] = 'localhost:8080'
os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = 'auth.json'
os.environ["PROJECT_ID"] = 'test_project'
os.environ["LOCATION"] = 'test_location'

import json
import sqlite3
import unittest
from unittest import mock
from unittest.mock import patch
from summarize import get_log_lines, fetch_from_db, get_summary
from wsclient import get_count_hours



TEST_LOG_LINES = [
                '{"envelope": {"timestamp": 1629878400000, "sourceName": "Author1", "sourceNumber": "+1234567890", "groupInfo": {"groupId": "abc123"}, "dataMessage": {"message": "Message1"}}}',
                '{"envelope": {"timestamp": 1629879000000, "sourceName": "Author2", "sourceNumber": "+1234567890", "groupInfo": {"groupId": "abc123"}, "dataMessage": {"message": "Message2"}}}',
                '{"envelope": {"timestamp": 1629879600000, "sourceName": "Author3", "sourceNumber": "+1234567890", "groupInfo": {"groupId": "abc123"}, "dataMessage": {"message": "Message3"}}}'
            ]
class TestGetLogLines(unittest.TestCase):
    def __init__(self, *args, **kwargs):
        super(TestGetLogLines, self).__init__(*args, **kwargs)
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
        

class TestFetchFromDB(unittest.TestCase):
    def test_fetch_from_db_count(self):
        # Mock the sqlite3 connection and cursor
        mock_cursor = unittest.mock.Mock()
        # This response gets reversed in the function
        mock_cursor.execute.return_value.fetchall.return_value = [('Author3: Message3'), ('Author2: Message2'), ('Author1: Message1')]
        mock_conn = unittest.mock.Mock()
        mock_conn.cursor.return_value = mock_cursor
        sqlite3.connect = unittest.mock.Mock(return_value=mock_conn)

        # Call the function with count parameter
        result = fetch_from_db('test_messages.db', count=3, starttime=None)

        # Assert the expected result
        expected_result = [('Author1: Message1'), ('Author2: Message2'), ('Author3: Message3')]
        self.assertEqual(result, expected_result)

    def test_fetch_from_db_hours(self):
        # Mock the sqlite3 connection and cursor
        mock_cursor = unittest.mock.Mock()
        mock_cursor.execute.return_value.fetchall.return_value = [('Author1: Message1'), ('Author2: Message2'), ('Author3: Message3')]
        mock_conn = unittest.mock.Mock()
        mock_conn.cursor.return_value = mock_cursor
        sqlite3.connect = unittest.mock.Mock(return_value=mock_conn)

        # Call the function with hours parameter
        result = fetch_from_db('test_messages.db', count=None, starttime=1629878400000)

        # Assert the expected result
        expected_result = [('Author1: Message1'), ('Author2: Message2'), ('Author3: Message3')]
        self.assertEqual(result, expected_result)

class TestGetLogLines(unittest.TestCase):
    @patch('summarize.fetch_from_db')
    def test_get_log_lines_with_hours(self, mock_fetch_from_db):
        # Mock the fetch_from_db function to return a list of messages
        mock_fetch_from_db.return_value = [('Author1: Message1',), ('Author2: Message2',), ('Author3: Message3',)]
        
        # Call the get_log_lines function with hours parameter
        result = get_log_lines('test_messages.db', hours=24, count=None, currenttime=1629878400000)
        
        # Assert the expected result
        expected_result = 'Author1: Message1\nAuthor2: Message2\nAuthor3: Message3'
        self.assertEqual(result, expected_result)
        mock_fetch_from_db.assert_called_once_with('test_messages.db', None, 1629792000000)
    
    @patch('summarize.fetch_from_db')
    def test_get_log_lines_with_count(self, mock_fetch_from_db):
        # Mock the fetch_from_db function to return a list of messages
        mock_fetch_from_db.return_value = [('Author1: Message1',), ('Author2: Message2',), ('Author3: Message3',)]
        
        # Call the get_log_lines function with count parameter
        result = get_log_lines('test_messages.db', hours=None, count=3, currenttime=1629878400000+604800000)
        
        # Assert the expected result
        expected_result = 'Author1: Message1\nAuthor2: Message2\nAuthor3: Message3'
        self.assertEqual(result, expected_result)
        mock_fetch_from_db.assert_called_once_with('test_messages.db', 3, 1629878400000)

    def test_get_log_lines_with_invalid_parameters(self):
        # Call the get_log_lines function with invalid parameters
        with self.assertRaises(ValueError):
            get_log_lines('test_messages.db', hours=None, count=None, currenttime=1629878400000)

    def test_get_log_lines_with_invalid_hours(self):
        # Call the get_log_lines function with invalid hours parameter
        with self.assertRaises(ValueError):
            get_log_lines('test_messages.db', hours=169, count=None, currenttime=1629878400000)

class TestGetSummary(unittest.TestCase):

    @patch('wsclient.get_count_hours')
    @patch('summarize.get_log_lines')
    @patch('summarize.summarize')
    def test_summary_request(self, mock_summarize, mock_get_log_lines, mock_get_count_hours):
        # Test when a summary request is made
        msg = "!summary 5h"
        mock_get_count_hours.return_value = (None, 5, None)
        mock_get_log_lines.return_value = "Log lines"
        mock_summarize.return_value = "Summary"

        summary = get_summary(5, None, None, 'test_messages.db')

        self.assertEqual(summary, "Summary")
        mock_get_log_lines.assert_called_once_with("test_messages.db", None, 5)
        mock_summarize.assert_called_once_with("Log lines", None)

    @patch('wsclient.get_count_hours')
    @patch('summarize.get_log_lines')
    @patch('summarize.summarize')
    def test_summary_request_with_question(self, mock_summarize, mock_get_log_lines, mock_get_count_hours):
        # Test when a summary request with a question is made
        question = "How are you?"
        mock_get_count_hours.return_value = (None, 168, question)
        mock_get_log_lines.return_value = "Log lines"
        mock_summarize.return_value = "Summary"

        summary = get_summary(None, 168, question, 'test_messages.db')

        self.assertEqual(summary, "Summary")
        mock_get_log_lines.assert_called_once_with("test_messages.db", 168, None)
        mock_summarize.assert_called_once_with("Log lines", "How are you?")

    @patch('wsclient.get_count_hours')
    @patch('summarize.get_log_lines')
    @patch('summarize.summarize')
    def test_summary_request_with_count(self, mock_summarize, mock_get_log_lines, mock_get_count_hours):
        # Test when a summary request with a count is made
        msg = "!summary 10"
        mock_get_count_hours.return_value = (10, 168, None)
        mock_get_log_lines.return_value = "Log lines"
        mock_summarize.return_value = "Summary"

        summary = get_summary(10, 168, None, 'test_messages.db')

        self.assertEqual(summary, "Summary")
        mock_get_log_lines.assert_called_once_with("test_messages.db", 168, 10)
        mock_summarize.assert_called_once_with("Log lines", None)

if __name__ == '__main__':
    unittest.main()