import json
import subprocess
from time import sleep

from constants import MANAGEMENT_HOST_NAME
from setup import get_config_from_file
from util import get_tunnel_connector_id

SINGLE_CASE_TIMEOUT = 600

class TunnellinkCli:
    def __init__(self, config, config_path, logger):
        self.basecmd = [config.tunnellink_binary, "tunnel"]
        if config_path is not None:
            self.basecmd += ["--config", str(config_path)]
        origincert = get_config_from_file()["origincert"]
        if origincert:
            self.basecmd += ["--origincert", origincert]
        self.logger = logger

    def _run_command(self, subcmd, subcmd_name, needs_to_pass=True):
        cmd = self.basecmd + subcmd
        # timeout limits the time a subprocess can run. This is useful to guard against running a tunnel when
        # command/args are in wrong order.
        result = run_subprocess(cmd, subcmd_name, self.logger, check=needs_to_pass, capture_output=True, timeout=15)
        return result

    def list_tunnels(self):
        cmd_args = ["list", "--output", "json"]
        listed = self._run_command(cmd_args, "list")
        return json.loads(listed.stdout)

    def get_management_token(self, config, config_path):
        basecmd = [config.tunnellink_binary]
        if config_path is not None:
            basecmd += ["--config", str(config_path)]
        origincert = get_config_from_file()["origincert"]
        if origincert:
            basecmd += ["--origincert", origincert]

        cmd_args = ["tail", "token", config.get_tunnel_id()]
        cmd = basecmd + cmd_args
        result = run_subprocess(cmd, "token", self.logger, check=True, capture_output=True, timeout=15)
        return json.loads(result.stdout.decode("utf-8").strip())["token"]
    
    def get_management_url(self, path, config, config_path):
        access_jwt = self.get_management_token(config, config_path)
        connector_id = get_tunnel_connector_id()
        return f"https://{MANAGEMENT_HOST_NAME}/{path}?connector_id={connector_id}&access_token={access_jwt}"
    
    def get_management_wsurl(self, path, config, config_path):
        access_jwt = self.get_management_token(config, config_path)
        connector_id = get_tunnel_connector_id()
        return f"wss://{MANAGEMENT_HOST_NAME}/{path}?connector_id={connector_id}&access_token={access_jwt}"

    def get_connector_id(self, config): 
        op = self.get_tunnel_info(config.get_tunnel_id())
        connectors = []
        for conn in op["conns"]:
            connectors.append(conn["id"])
        return connectors

    def get_tunnel_info(self, tunnel_id):
        info = self._run_command(["info", "--output", "json", tunnel_id], "info")
        return json.loads(info.stdout)

    def __enter__(self):
        self.basecmd += ["run"]
        self.process = subprocess.Popen(self.basecmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.logger.info(f"Run cmd {self.basecmd}")
        return self.process

    def __exit__(self, exc_type, exc_value, exc_traceback):
        terminate_gracefully(self.process, self.logger, self.basecmd)
        self.logger.debug(f"{self.basecmd} logs: {self.process.stderr.read()}")


def terminate_gracefully(process, logger, cmd):
    process.terminate()
    process_terminated = wait_for_terminate(process)
    if not process_terminated:
        process.kill()
        logger.warning(f"{cmd}: tunnellink did not terminate within wait period. Killing process. logs: \
                stdout: {process.stdout.read()}, stderr: {process.stderr.read()}")


def wait_for_terminate(opened_subprocess, attempts=10, poll_interval=1):
    """
        wait_for_terminate polls the opened_subprocess every x seconds for a given number of attempts.
        It returns true if the subprocess was terminated and false if it didn't.
    """
    for _ in range(attempts):
        if _is_process_stopped(opened_subprocess):
            return True
        sleep(poll_interval)
    return False


def _is_process_stopped(process):
    return process.poll() is not None


def cert_path():
    return get_config_from_file()["origincert"]


class SubprocessError(Exception):
    def __init__(self, program, exit_code, cause):
        self.program = program
        self.exit_code = exit_code
        self.cause = cause


def run_subprocess(cmd, cmd_name, logger, timeout=SINGLE_CASE_TIMEOUT, **kargs):
    kargs["timeout"] = timeout
    try:
        result = subprocess.run(cmd, **kargs)
        logger.debug(f"{cmd} log: {result.stdout}", extra={"cmd": cmd_name})
        return result
    except subprocess.CalledProcessError as e:
        err = f"{cmd} return exit code {e.returncode}, stderr" + e.stderr.decode("utf-8")
        logger.error(err, extra={"cmd": cmd_name, "return_code": e.returncode})
        raise SubprocessError(cmd[0], e.returncode, e)
    except subprocess.TimeoutExpired as e:
        err = f"{cmd} timeout after {e.timeout} seconds, stdout: {e.stdout}, stderr: {e.stderr}"
        logger.error(err, extra={"cmd": cmd_name, "return_code": "timeout"})
        raise e