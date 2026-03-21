import unittest
from unittest.mock import Mock, patch

from src import __main__ as entrypoint


class TestEntryPoint(unittest.TestCase):
    @patch.object(entrypoint, 'run_server')
    @patch.object(entrypoint.threading, 'Thread')
    def test_main_disables_reloader_in_combined_mode(self, thread_cls, run_server):
        thread = Mock()
        thread_cls.return_value = thread

        entrypoint.main(['scrape', 'server'])

        thread_cls.assert_called_once_with(
            target=entrypoint.run_async_scrape,
            daemon=True,
            name='scrape-loop',
        )
        thread.start.assert_called_once()
        run_server.assert_called_once_with(use_reloader=False)

    @patch.object(entrypoint, 'run_server')
    def test_main_keeps_server_only_path_unchanged(self, run_server):
        entrypoint.main(['server'])

        run_server.assert_called_once_with()


if __name__ == '__main__':
    unittest.main()
