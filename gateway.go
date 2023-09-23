package iotgateway

import (
	"context"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gookit/event"
	"github.com/sagoo-cloud/iotgateway/conf"
	"github.com/sagoo-cloud/iotgateway/lib"
	"github.com/sagoo-cloud/iotgateway/log"
	"github.com/sagoo-cloud/iotgateway/model"
	"github.com/sagoo-cloud/iotgateway/mqttClient"
	"github.com/sagoo-cloud/iotgateway/mqttProtocol"
	"github.com/sagoo-cloud/iotgateway/network"
	"github.com/sagoo-cloud/iotgateway/vars"
	"strings"
	"time"
)

const (
	propertyTopic = "/sys/%s/%s/thing/event/property/pack/post"
	serviceTopic  = "/sys/+/%s/thing/service/#"
)

type gateway struct {
	Address    string
	Version    string
	Status     string
	ctx        context.Context // 上下文
	options    *conf.GatewayConfig
	MQTTClient mqtt.Client
	Server     *network.TcpServer
	Protocol   mqttProtocol.Protocol
}

var ServerGateway *gateway

func NewGateway(options *conf.GatewayConfig, protocol mqttProtocol.Protocol) (g *gateway, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mqttClient, err := mqttClient.GetMQTTClient(options.MqttConfig) //初始化mqtt客户端
	tcpServer := network.NewServer(options.GatewayServerConfig.Addr)
	vars.GatewayServerConfig = options.GatewayServerConfig
	if err != nil {
		log.Debug("mqttClient.GetMQTTClient error:", err)
	}
	g = &gateway{
		options:    options,
		Address:    options.GatewayServerConfig.Addr,
		ctx:        ctx,
		MQTTClient: mqttClient, // will be set later
		Server:     tcpServer,
		Protocol:   protocol,
	}
	ServerGateway = g
	//g.heartbeat(ctx, options.GatewayServerConfig.Duration)
	return
}
func (g *gateway) Start() error {
	log.Info("TCPServer started listening on %s", g.Address)
	g.Server.Start(g.ctx, g.Protocol)
	return nil
}

// SubscribeEvent  订阅平台的服务调用
func (g *gateway) SubscribeEvent(deviceKey string) {
	if g.MQTTClient == nil || !g.MQTTClient.IsConnected() {
		log.Error("Client has lost connection with the MQTT broker.")
		return
	}
	topic := fmt.Sprintf(serviceTopic, deviceKey)
	log.Debug("topic: ", topic)
	token := g.MQTTClient.Subscribe(topic, 1, onMessage)
	if token.Error() != nil {
		log.Debug("subscribe error: ", token.Error())
	}
}

var onMessage mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	//忽略_reply结尾的topic
	if strings.HasSuffix(msg.Topic(), "_reply") {
		return
	}
	if msg != nil {
		//通过监听到的topic地址获取设备标识
		deviceKey := lib.GetTopicInfo("deviceKey", msg.Topic())
		var data = mqttProtocol.ServiceCallRequest{}
		log.Debug("==111==收到服务下发的topic====", msg.Topic())
		log.Debug("====收到服务下发的信息====", msg.Payload())

		err := gconv.Scan(msg.Payload(), &data)
		if err != nil {
			log.Debug("解析服务功能数据出错： %s", err)
			return
		}

		//触发下发事件
		data.Params["DeviceKey"] = deviceKey

		method := strings.Split(data.Method, ".")
		var up model.UpMessage
		up.MessageID = data.Id
		up.SendTime = time.Now().UnixNano() / 1e9
		up.MethodName = method[2]
		up.Topic = msg.Topic()
		vars.UpdateUpMessageMap(deviceKey, up)
		ra, ee := vars.GetUpMessageMap(deviceKey)
		log.Debug("==222===MessageHandler===========", ra, ee)
		event.MustFire(method[2], data.Params)
	}
}
