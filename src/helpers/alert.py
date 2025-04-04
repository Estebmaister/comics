import os
import smtplib
import subprocess
from typing import Dict, List, Optional

import helpers.logger
from db import Publishers

# Configure logging
log = helpers.logger.get_logger(__name__)

# Constants
SMTP_SERVER = 'smtp.gmail.com'
SMTP_PORT = 587
ALERT_ICON_PATH = '/usr/share/icons/Adwaita/scalable/status/software-update-urgent-symbolic.svg'

# Email configuration
EMAIL = os.getenv('EMAIL', 'estebmaister@gmail.com')
EMAIL_PASSWORD = os.getenv('EMAIL_PASS', '')

# Global state
alert_state: Dict[str, any] = {
    'title': 'Scrape alert',
    'content': '',
    'alert_count': 0
}


def send_email(subject: str, body: str) -> None:
    """
    Send an email notification using Gmail SMTP.

    Args:
        subject: Email subject line
        body: Email body content

    Raises:
        smtplib.SMTPException: If email sending fails
    """
    try:
        with smtplib.SMTP(SMTP_SERVER, SMTP_PORT) as server:
            server.starttls()
            server.login(EMAIL, EMAIL_PASSWORD)

            message = f"Subject: {subject}\n\n{body}"
            server.sendmail(EMAIL, EMAIL, message.encode('ascii', 'ignore'))
    except smtplib.SMTPException as e:
        log.error('Failed to send email: %s', str(e))
        # example: (421, b'4.4.5 Server busy, try again later.')
        raise


def send_reminder() -> None:
    """
    Send notifications if there are pending alerts.
    This will send both desktop and email notifications if alerts exist.
    """
    if alert_state['alert_count'] == 0:
        return

    notification_title = f"{alert_state['title']} - ({alert_state['alert_count']})"

    try:
        send_email(notification_title, alert_state['content'])
    except Exception as e:
        # Continue, don't reset alert state
        return

    # Reset alert state
    alert_state['alert_count'] = 0
    alert_state['content'] = ''


def add_alert(title: str, chapter: str, publishers: List[Publishers]) -> None:
    """
    Add a new alert for a comic update.

    Args:
        title: Comic title
        chapter: Chapter number or identifier
        publishers: List of publishers for this comic
    """
    publisher_names = [Publishers(pub).name for pub in publishers]
    update_message = f'{title}, ch {chapter} - {publisher_names}'

    helpers.logger.update(update_message)

    alert_state['alert_count'] += 1
    alert_state['content'] += f'\n{update_message}'

    # Send reminder if we've accumulated enough alerts
    if alert_state['alert_count'] >= 5:
        send_reminder()
