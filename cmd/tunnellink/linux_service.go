//go:build linux

package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"

	"github.com/khulnasoft/tunnellink/cmd/tunnellink/cliutil"
	"github.com/khulnasoft/tunnellink/cmd/tunnellink/tunnel"
	"github.com/khulnasoft/tunnellink/config"
	"github.com/khulnasoft/tunnellink/logger"
)

func runApp(app *cli.App, graceShutdownC chan struct{}) {
	app.Commands = append(app.Commands, &cli.Command{
		Name:  "service",
		Usage: "Manages the tunnellink system service",
		Subcommands: []*cli.Command{
			{
				Name:   "install",
				Usage:  "Install tunnellink as a system service",
				Action: cliutil.ConfiguredAction(installLinuxService),
				Flags: []cli.Flag{
					noUpdateServiceFlag,
				},
			},
			{
				Name:   "uninstall",
				Usage:  "Uninstall the tunnellink service",
				Action: cliutil.ConfiguredAction(uninstallLinuxService),
			},
		},
	})
	app.Run(os.Args)
}

// The directory and files that are used by the service.
// These are hard-coded in the templates below.
const (
	serviceConfigDir        = "/etc/tunnellink"
	serviceConfigFile       = "config.yml"
	serviceCredentialFile   = "cert.pem"
	serviceConfigPath       = serviceConfigDir + "/" + serviceConfigFile
	tunnellinkService       = "tunnellink.service"
	tunnellinkUpdateService = "tunnellink-update.service"
	tunnellinkUpdateTimer   = "tunnellink-update.timer"
)

var systemdAllTemplates = map[string]ServiceTemplate{
	tunnellinkService: {
		Path: fmt.Sprintf("/etc/systemd/system/%s", tunnellinkService),
		Content: `[Unit]
Description=tunnellink
After=network-online.target
Wants=network-online.target

[Service]
TimeoutStartSec=0
Type=notify
ExecStart={{ .Path }} --no-autoupdate{{ range .ExtraArgs }} {{ . }}{{ end }}
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
`,
	},
	tunnellinkUpdateService: {
		Path: fmt.Sprintf("/etc/systemd/system/%s", tunnellinkUpdateService),
		Content: `[Unit]
Description=Update tunnellink
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/bin/bash -c '{{ .Path }} update; code=$?; if [ $code -eq 11 ]; then systemctl restart tunnellink; exit 0; fi; exit $code'
`,
	},
	tunnellinkUpdateTimer: {
		Path: fmt.Sprintf("/etc/systemd/system/%s", tunnellinkUpdateTimer),
		Content: `[Unit]
Description=Update tunnellink

[Timer]
OnCalendar=daily

[Install]
WantedBy=timers.target
`,
	},
}

var sysvTemplate = ServiceTemplate{
	Path:     "/etc/init.d/tunnellink",
	FileMode: 0755,
	Content: `#!/bin/sh
# For RedHat and cousins:
# chkconfig: 2345 99 01
# description: tunnellink
# processname: {{.Path}}
### BEGIN INIT INFO
# Provides:          {{.Path}}
# Required-Start:
# Required-Stop:
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: tunnellink
# Description:       tunnellink agent
### END INIT INFO
name=$(basename $(readlink -f $0))
cmd="{{.Path}} --pidfile /var/run/$name.pid {{ range .ExtraArgs }} {{ . }}{{ end }}"
pid_file="/var/run/$name.pid"
stdout_log="/var/log/$name.log"
stderr_log="/var/log/$name.err"
[ -e /etc/sysconfig/$name ] && . /etc/sysconfig/$name
get_pid() {
    cat "$pid_file"
}
is_running() {
    [ -f "$pid_file" ] && ps $(get_pid) > /dev/null 2>&1
}
case "$1" in
    start)
        if is_running; then
            echo "Already started"
        else
            echo "Starting $name"
            $cmd >> "$stdout_log" 2>> "$stderr_log" &
            echo $! > "$pid_file"
        fi
    ;;
    stop)
        if is_running; then
            echo -n "Stopping $name.."
            kill $(get_pid)
            for i in {1..10}
            do
                if ! is_running; then
                    break
                fi
                echo -n "."
                sleep 1
            done
            echo
            if is_running; then
                echo "Not stopped; may still be shutting down or shutdown may have failed"
                exit 1
            else
                echo "Stopped"
                if [ -f "$pid_file" ]; then
                    rm "$pid_file"
                fi
            fi
        else
            echo "Not running"
        fi
    ;;
    restart)
        $0 stop
        if is_running; then
            echo "Unable to stop, will not attempt to start"
            exit 1
        fi
        $0 start
    ;;
    status)
        if is_running; then
            echo "Running"
        else
            echo "Stopped"
            exit 1
        fi
    ;;
    *)
    echo "Usage: $0 {start|stop|restart|status}"
    exit 1
    ;;
esac
exit 0
`,
}

var (
	noUpdateServiceFlag = &cli.BoolFlag{
		Name:  "no-update-service",
		Usage: "Disable auto-update of the tunnellink linux service, which restarts the server to upgrade for new versions.",
		Value: false,
	}
)

func isSystemd() bool {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return true
	}
	return false
}

func installLinuxService(c *cli.Context) error {
	log := logger.CreateLoggerFromContext(c, logger.EnableTerminalLog)

	etPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error determining executable path: %v", err)
	}
	templateArgs := ServiceTemplateArgs{
		Path: etPath,
	}

	// Check if the "no update flag" is set
	autoUpdate := !c.IsSet(noUpdateServiceFlag.Name)

	var extraArgsFunc func(c *cli.Context, log *zerolog.Logger) ([]string, error)
	if c.NArg() == 0 {
		extraArgsFunc = buildArgsForConfig
	} else {
		extraArgsFunc = buildArgsForToken
	}

	extraArgs, err := extraArgsFunc(c, log)
	if err != nil {
		return err
	}

	templateArgs.ExtraArgs = extraArgs

	switch {
	case isSystemd():
		log.Info().Msgf("Using Systemd")
		err = installSystemd(&templateArgs, autoUpdate, log)
	default:
		log.Info().Msgf("Using SysV")
		err = installSysv(&templateArgs, autoUpdate, log)
	}

	if err == nil {
		log.Info().Msg("Linux service for tunnellink installed successfully")
	}
	return err
}

func buildArgsForConfig(c *cli.Context, log *zerolog.Logger) ([]string, error) {
	if err := ensureConfigDirExists(serviceConfigDir); err != nil {
		return nil, err
	}

	src, _, err := config.ReadConfigFile(c, log)
	if err != nil {
		return nil, err
	}

	// can't use context because this command doesn't define "credentials-file" flag
	configPresent := func(s string) bool {
		val, err := src.String(s)
		return err == nil && val != ""
	}
	if src.TunnelID == "" || !configPresent(tunnel.CredFileFlag) {
		return nil, fmt.Errorf(`Configuration file %s must contain entries for the tunnel to run and its associated credentials:
tunnel: TUNNEL-UUID
credentials-file: CREDENTIALS-FILE
`, src.Source())
	}
	if src.Source() != serviceConfigPath {
		if exists, err := config.FileExists(serviceConfigPath); err != nil || exists {
			return nil, fmt.Errorf("Possible conflicting configuration in %[1]s and %[2]s. Either remove %[2]s or run `tunnellink --config %[2]s service install`", src.Source(), serviceConfigPath)
		}

		if err := copyFile(src.Source(), serviceConfigPath); err != nil {
			return nil, fmt.Errorf("failed to copy %s to %s: %w", src.Source(), serviceConfigPath, err)
		}
	}

	return []string{
		"--config", "/etc/tunnellink/config.yml", "tunnel", "run",
	}, nil
}

func installSystemd(templateArgs *ServiceTemplateArgs, autoUpdate bool, log *zerolog.Logger) error {
	var systemdTemplates []ServiceTemplate
	if autoUpdate {
		systemdTemplates = []ServiceTemplate{
			systemdAllTemplates[tunnellinkService],
			systemdAllTemplates[tunnellinkUpdateService],
			systemdAllTemplates[tunnellinkUpdateTimer],
		}
	} else {
		systemdTemplates = []ServiceTemplate{
			systemdAllTemplates[tunnellinkService],
		}
	}

	for _, serviceTemplate := range systemdTemplates {
		err := serviceTemplate.Generate(templateArgs)
		if err != nil {
			log.Err(err).Msg("error generating service template")
			return err
		}
	}
	if err := runCommand("systemctl", "enable", tunnellinkService); err != nil {
		log.Err(err).Msgf("systemctl enable %s error", tunnellinkService)
		return err
	}

	if autoUpdate {
		if err := runCommand("systemctl", "start", tunnellinkUpdateTimer); err != nil {
			log.Err(err).Msgf("systemctl start %s error", tunnellinkUpdateTimer)
			return err
		}
	}

	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		log.Err(err).Msg("systemctl daemon-reload error")
		return err
	}
	return runCommand("systemctl", "start", tunnellinkService)
}

func installSysv(templateArgs *ServiceTemplateArgs, autoUpdate bool, log *zerolog.Logger) error {
	confPath, err := sysvTemplate.ResolvePath()
	if err != nil {
		log.Err(err).Msg("error resolving system path")
		return err
	}

	if autoUpdate {
		templateArgs.ExtraArgs = append([]string{"--autoupdate-freq 24h0m0s"}, templateArgs.ExtraArgs...)
	} else {
		templateArgs.ExtraArgs = append([]string{"--no-autoupdate"}, templateArgs.ExtraArgs...)
	}

	if err := sysvTemplate.Generate(templateArgs); err != nil {
		log.Err(err).Msg("error generating system template")
		return err
	}
	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err := os.Symlink(confPath, "/etc/rc"+i+".d/S50et"); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err := os.Symlink(confPath, "/etc/rc"+i+".d/K02et"); err != nil {
			continue
		}
	}
	return runCommand("service", "tunnellink", "start")
}

func uninstallLinuxService(c *cli.Context) error {
	log := logger.CreateLoggerFromContext(c, logger.EnableTerminalLog)

	var err error
	switch {
	case isSystemd():
		log.Info().Msg("Using Systemd")
		err = uninstallSystemd(log)
	default:
		log.Info().Msg("Using SysV")
		err = uninstallSysv(log)
	}

	if err == nil {
		log.Info().Msg("Linux service for tunnellink uninstalled successfully")
	}
	return err
}

func uninstallSystemd(log *zerolog.Logger) error {
	// Get only the installed services
	installedServices := make(map[string]ServiceTemplate)
	for serviceName, serviceTemplate := range systemdAllTemplates {
		if err := runCommand("systemctl", "list-units", "--all", "|", "grep", serviceName); err == nil {
			installedServices[serviceName] = serviceTemplate
		} else {
			log.Info().Msgf("Service '%s' not installed, skipping its uninstall", serviceName)
		}
	}

	if _, exists := installedServices[tunnellinkService]; exists {
		if err := runCommand("systemctl", "disable", tunnellinkService); err != nil {
			log.Err(err).Msgf("systemctl disable %s error", tunnellinkService)
			return err
		}
		if err := runCommand("systemctl", "stop", tunnellinkService); err != nil {
			log.Err(err).Msgf("systemctl stop %s error", tunnellinkService)
			return err
		}
	}

	if _, exists := installedServices[tunnellinkUpdateTimer]; exists {
		if err := runCommand("systemctl", "stop", tunnellinkUpdateTimer); err != nil {
			log.Err(err).Msgf("systemctl stop %s error", tunnellinkUpdateTimer)
			return err
		}
	}

	for _, serviceTemplate := range installedServices {
		if err := serviceTemplate.Remove(); err != nil {
			log.Err(err).Msg("error removing service template")
			return err
		}
	}
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		log.Err(err).Msg("systemctl daemon-reload error")
		return err
	}
	return nil
}

func uninstallSysv(log *zerolog.Logger) error {
	if err := runCommand("service", "tunnellink", "stop"); err != nil {
		log.Err(err).Msg("service tunnellink stop error")
		return err
	}
	if err := sysvTemplate.Remove(); err != nil {
		log.Err(err).Msg("error removing service template")
		return err
	}
	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err := os.Remove("/etc/rc" + i + ".d/S50et"); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err := os.Remove("/etc/rc" + i + ".d/K02et"); err != nil {
			continue
		}
	}
	return nil
}
