package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/google/nftables/expr"
	"github.com/routesentry/routesentry/pkg/firewall"
	"github.com/routesentry/routesentry/pkg/routing"
	"go.uber.org/zap"
)

const (
	gwIPEnvKey   = "GATEWAY_IP"
	gwPortEnvKey = "GATEWAY_PORT"
)

var (
	ethIface = GetEnvOrDefault("OIFName", "eth0")
)

func GetEnv(name string) (string, error) {
	val, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("missing required environment variable: %s", name)
	}
	if val == "" {
		return "", fmt.Errorf("environment variable %s is empty", name)
	}
	return val, nil
}

func GetEnvOrDefault(name string, defaultVal string) string {
	if val := os.Getenv(name); val != "" {
		return val
	}
	return defaultVal
}

func LookupUDPAddr(ip string, port string) (*net.UDPAddr, error) {
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ip, port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %s", err)
	}
	return addr, nil
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil && !errors.Is(err, syscall.EINVAL) {
			log.Fatalf("Failed to sync logger: %v", err)
		}
	}(logger)

	gwIP, err := GetEnv(gwIPEnvKey)
	if err != nil {
		logger.Fatal("Failed to get gateway IP", zap.Error(err))
	}
	gwPort, err := GetEnv(gwPortEnvKey)
	if err != nil {
		logger.Fatal("Failed to get gateway port", zap.Error(err))
	}

	gwAddr, err := LookupUDPAddr(gwIP, gwPort)
	if err != nil {
		logger.Fatal("Failed to lookup gateway address", zap.Error(err))
	}

	hrLogger := logger.With(zap.String("to", gwAddr.IP.String()), zap.String("via", ethIface))
	hrLogger.Info("Adding HostRoute...")
	err = routing.AddHostRoute(ethIface, gwAddr.IP)
	if err != nil {
		if !errors.Is(err, syscall.EEXIST) {
			hrLogger.Fatal("Failed to add host route", zap.Error(err))
		} else {
			hrLogger.Info("Route already exists")
		}
	}
	hrLogger.Info("Successfully added HostRoute")

	f, err := firewall.New()
	if err != nil {
		logger.Fatal("Failed to create firewall.", zap.Error(err))
	}

	logger.Info("Configuring blackhole firewall...")
	err = ConfigureFirewallBlackHole(f)
	if err != nil {
		logger.Fatal("Failed to enable kill switch", zap.Error(err))
	}
	logger.Info("Firewall is enabled.")
}

const (
	LoopbackIfaceName = "lo"
)

// ConfigureFirewallBlackHole sets up nftables to block all traffic except for loopback
// out on loopback device
// out on handshake traffic
// in  on existing connections
func ConfigureFirewallBlackHole(f *firewall.Firewall) error {

	loopBackRule := f.NewRuleBuilder(firewall.Output).
		MatchMetaOIFName(LoopbackIfaceName).
		Verdict(expr.VerdictAccept).
		Build()
	f.AddRule(loopBackRule)
	//
	//tunRule := f.NewRuleBuilder(firewall.Output).
	//	MatchMetaOIFName(gatewayIfaceName).
	//	Verdict(expr.VerdictAccept).
	//	Build()
	//f.AddRule(tunRule)
	//
	//handshakeRule := f.NewRuleBuilder(firewall.Output).
	//	MatchMetaOIFName(egressIfaceName).
	//	MatchL4Proto(firewall.UDP).
	//	MatchDestinationIP(gatewayAddr.IP).
	//	MatchUDPDestPort(uint16(gatewayAddr.Port)).
	//	Verdict(expr.VerdictAccept).
	//	Build()
	//f.AddRule(handshakeRule)

	//maskedStates := expr.CtStateBitESTABLISHED | expr.CtStateBitRELATED
	//establishedRule := f.NewRuleBuilder(firewall.Input).
	//	CtStateIn(uint8(maskedStates)).
	//	Verdict(expr.VerdictAccept).
	//	Build()
	//f.AddRule(establishedRule)

	if err := f.Flush(); err != nil {
		return fmt.Errorf("error while applying firewall rules: %w", err)
	}

	return nil
}
