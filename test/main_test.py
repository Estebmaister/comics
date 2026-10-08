import unittest
from unittest.mock import patch

from src import __main__ as entrypoint


class TestEntryPoint(unittest.TestCase):
    @patch.object(entrypoint, 'run_server')
    def test_main_server_only(self, run_server):
        entrypoint.main(['server'])

        run_server.assert_called_once_with()

    @patch.object(entrypoint.sys, 'exit')
    def test_main_scrape_flag_points_to_go(self, exit_mock):
        entrypoint.main(['scrape'])
        exit_mock.assert_called_once_with(2)


if __name__ == '__main__':
    unittest.main()
