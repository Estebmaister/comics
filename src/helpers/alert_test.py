import smtplib
import subprocess
from unittest.mock import MagicMock, call, patch

import pytest

from helpers.alert import (ALERT_ICON_PATH, DEFAULT_ALERT_MESSAGE, add_alert,
                           alert_state, send_email, send_reminder)


@pytest.fixture
def reset_alert_state():
    """Reset alert state before each test"""
    alert_state['alert_count'] = 0
    alert_state['content'] = DEFAULT_ALERT_MESSAGE
    alert_state['title'] = 'Scrape alert'
    yield
    # Clean up after test
    alert_state['alert_count'] = 0
    alert_state['content'] = DEFAULT_ALERT_MESSAGE
    alert_state['title'] = 'Scrape alert'


@pytest.mark.parametrize("subject,body", [
    ("Test Subject", "Test Body"),
    ("Alert", "Important Message"),
    ("", "Empty Subject Test"),
])
def test_send_email(subject, body):
    """Test email sending with various inputs"""
    mock_server = MagicMock()

    with patch('smtplib.SMTP') as mock_smtp:
        mock_smtp.return_value.__enter__.return_value = mock_server
        send_email(subject, body)

        mock_server.starttls.assert_called_once()
        mock_server.login.assert_called_once()
        mock_server.sendmail.assert_called_once()


def test_send_email_handles_smtp_error():
    """Test handling of SMTP errors"""
    with patch('smtplib.SMTP') as mock_smtp, \
            patch('helpers.alert.log.error') as mock_log:
        mock_smtp.return_value.__enter__.side_effect = smtplib.SMTPException()

        with pytest.raises(smtplib.SMTPException):
            send_email("Test", "Content")

        mock_log.assert_called_once()


def test_add_alert(reset_alert_state):
    """Test adding a new alert"""
    publishers = [1, 2]  # Using dummy publisher IDs

    with patch('helpers.alert.send_reminder') as mock_reminder:
        add_alert("Test Comic", "123", publishers)

        assert alert_state['alert_count'] == 1
        assert "Test Comic" in alert_state['content']
        assert "ch <b>123</b>" in alert_state['content']
        mock_reminder.assert_not_called()


def test_add_alert_triggers_reminder(reset_alert_state):
    """Test that reminder is triggered after 4 alerts"""
    publishers = [1]

    with patch('helpers.alert.send_reminder') as mock_reminder:
        # Add 4 alerts
        for i in range(4):
            add_alert(f"Comic {i}", str(i), publishers)

        assert alert_state['alert_count'] == 4
        mock_reminder.assert_called_once()


if __name__ == '__main__':
    pytest.main([__file__])
