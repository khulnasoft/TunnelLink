#!/usr/bin/env python
from contextlib import contextmanager
import platform
import signal
import threading
import time

import pytest
import requests

from constants import protocols
from util import start_tunnellink, wait_tunnel_ready, check_tunnel_not_connected


def supported_signals():
    if platform.system() == "Windows":
        return [signal.SIGTERM]
    return [signal.SIGTERM, signal.SIGINT]


class TestTermination:
    grace_period = 5
    timeout = 10
    sse_endpoint = "/sse?freq=1s"

    def _extra_config(self, protocol):
        return {
            "grace-period": f"{self.grace_period}s",
            "protocol": protocol,
        }

    @pytest.mark.parametrize("signal", supported_signals())
    @pytest.mark.parametrize("protocol", protocols())
    def test_graceful_shutdown(self, tmp_path, component_tests_config, signal, protocol):
        config = component_tests_config(self._extra_config(protocol))
        with start_tunnellink(
                tmp_path, config, cfd_pre_args=["tunnel", "--ha-connections", "1"],  new_process=True, capture_output=False) as tunnellink:
            wait_tunnel_ready(tunnel_url=config.get_url())

            connected = threading.Condition()
            in_flight_req = threading.Thread(
                target=self.stream_request, args=(config, connected, False, ))
            in_flight_req.start()

            with connected:
                connected.wait(self.timeout)
            # Send signal after the SSE connection is established
            with self.within_grace_period():
                self.terminate_by_signal(tunnellink, signal)
                self.wait_eyeball_thread(
                    in_flight_req, self.grace_period + self.timeout)

    # test tunnellink terminates before grace period expires when all eyeball
    # connections are drained
    @pytest.mark.parametrize("signal", supported_signals())
    @pytest.mark.parametrize("protocol", protocols())
    def test_shutdown_once_no_connection(self, tmp_path, component_tests_config, signal, protocol):
        config = component_tests_config(self._extra_config(protocol))
        with start_tunnellink(
                tmp_path, config, cfd_pre_args=["tunnel", "--ha-connections", "1"], new_process=True, capture_output=False) as tunnellink:
            wait_tunnel_ready(tunnel_url=config.get_url())

            connected = threading.Condition()
            in_flight_req = threading.Thread(
                target=self.stream_request, args=(config, connected, True, ))
            in_flight_req.start()

            with connected:
                connected.wait(self.timeout)
            with self.within_grace_period(has_connection=False):
                # Send signal after the SSE connection is established
                self.terminate_by_signal(tunnellink, signal)
                self.wait_eyeball_thread(in_flight_req, self.grace_period)

    @pytest.mark.parametrize("signal", supported_signals())
    @pytest.mark.parametrize("protocol", protocols())
    def test_no_connection_shutdown(self, tmp_path, component_tests_config, signal, protocol):
        config = component_tests_config(self._extra_config(protocol))
        with start_tunnellink(
                tmp_path, config, cfd_pre_args=["tunnel", "--ha-connections", "1"], new_process=True, capture_output=False) as tunnellink:
            wait_tunnel_ready(tunnel_url=config.get_url())
            with self.within_grace_period(has_connection=False):
                self.terminate_by_signal(tunnellink, signal)

    def terminate_by_signal(self, tunnellink, sig):
        tunnellink.send_signal(sig)
        check_tunnel_not_connected()
        tunnellink.wait()

    def wait_eyeball_thread(self, thread, timeout):
        thread.join(timeout)
        assert thread.is_alive() == False, "eyeball thread is still alive"

    # Using this context asserts logic within the context is executed within grace period
    @contextmanager
    def within_grace_period(self, has_connection=True):
        try:
            start = time.time()
            yield
        finally:

            # If the request takes longer than the grace period then we need to wait at most the grace period.
            # If the request fell within the grace period tunnellink can close earlier, but to ensure that it doesn't
            # close immediately we add a minimum boundary. If tunnellink shutdown in less than 1s it's likely that
            # it shutdown as soon as it received SIGINT. The only way tunnellink can close immediately is if it has no
            # in-flight requests
            minimum = 1 if has_connection else 0
            duration = time.time() - start
            # Here we truncate to ensure that we don't fail on minute differences like 10.1 instead of 10
            assert minimum <= int(duration) <= self.grace_period

    def stream_request(self, config, connected, early_terminate):
        expected_terminate_message = "502 Bad Gateway"
        url = config.get_url() + self.sse_endpoint

        with requests.get(url, timeout=5, stream=True) as resp:
            with connected:
                connected.notifyAll()
            lines = 0
            for line in resp.iter_lines():
                if expected_terminate_message.encode() == line:
                    break
                lines += 1
                if early_terminate and lines == 2:
                    return
            # /sse returns count followed by 2 new lines
            assert lines >= (self.grace_period * 2)
