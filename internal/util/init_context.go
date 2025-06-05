package util

import (
	"context"
	"net"
	"net/netip"
	"os"

	"github.com/google/uuid"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	pcf_context "github.com/free5gc/pcf/internal/context"
	"github.com/free5gc/pcf/internal/logger"
	"github.com/free5gc/pcf/pkg/factory"
	"github.com/free5gc/util/mongoapi"
)

// Init PCF Context from config flie
func InitpcfContext(context *pcf_context.PCFContext) {
	config := factory.PcfConfig
	logger.UtilLog.Infof("pcfconfig Info: Version[%s] Description[%s]", config.Info.Version, config.Info.Description)
	configuration := config.Configuration
	context.NfId = uuid.New().String()
	if configuration.PcfName != "" {
		context.Name = configuration.PcfName
	}

	mongodb := config.Configuration.Mongodb
	// Connect to MongoDB
	if err := mongoapi.SetMongoDB(mongodb.Name, mongodb.Url); err != nil {
		logger.UtilLog.Errorf("InitpcfContext err: %+v", err)
		return
	}

	sbi := configuration.Sbi
	context.NrfUri = configuration.NrfUri

	context.SBIPort = sbi.Port

	context.UriScheme = models.UriScheme(sbi.Scheme)

	if bindingIP := os.Getenv(sbi.BindingIP); bindingIP != "" {
		logger.UtilLog.Info("Parsing BindingIP address from ENV Variable.")
		sbi.BindingIP = bindingIP
	}
	if registerIP := os.Getenv(sbi.RegisterIP); registerIP != "" {
		logger.UtilLog.Info("Parsing RegisterIP address from ENV Variable.")
		sbi.RegisterIP = registerIP
	}
	context.BindingIP = resolveIP(sbi.BindingIP)
	context.RegisterIP = resolveIP(sbi.RegisterIP)

	serviceList := configuration.ServiceList
	context.InitNFService(serviceList, config.Info.Version)
	context.TimeFormat = configuration.TimeFormat
	context.DefaultBdtRefId = configuration.DefaultBdtRefId
	for _, service := range context.NfService {
		var err error
		context.PcfServiceUris[service.ServiceName] = service.ApiPrefix +
			"/" + string(service.ServiceName) + "/" + (service.Versions)[0].ApiVersionInUri
		context.PcfSuppFeats[service.ServiceName], err = openapi.NewSupportedFeature(service.SupportedFeatures)
		if err != nil {
			logger.UtilLog.Errorf("openapi NewSupportedFeature error: %+v", err)
		}
	}
	context.Locality = configuration.Locality
}

func resolveIP(ip string) netip.Addr {
	resolvedIPs, err := net.DefaultResolver.LookupNetIP(context.Background(), "ip", ip)
	if err != nil {
		logger.InitLog.Errorf("Lookup failed with %s: %+v", ip, err)
	}
	resolvedIP := resolvedIPs[0].Unmap()
	if resolvedIP := resolvedIP.String(); resolvedIP != ip {
		logger.UtilLog.Infof("Lookup revolved %s into %s", ip, resolvedIP)
	}
	return resolvedIP
}

func GetIpEndPoint(context *pcf_context.PCFContext) []models.IpEndPoint {
	if context.RegisterIP.Is6() {
		return []models.IpEndPoint{
			{
				Ipv6Address: context.RegisterIP.String(),
				Transport:   models.NrfNfManagementTransportProtocol_TCP,
				Port:        int32(context.SBIPort),
			},
		}
	} else if context.RegisterIP.Is4() {
		return []models.IpEndPoint{
			{
				Ipv4Address: context.RegisterIP.String(),
				Transport:   models.NrfNfManagementTransportProtocol_TCP,
				Port:        int32(context.SBIPort),
			},
		}
	}
	return nil
}
