# eemqtt 组件使用指南
[![goproxy.cn](https://goproxy.cn/stats/github.com/ego-component/eemqtt/badges/download-count.svg)](https://goproxy.cn/stats/github.com/ego-component/eemqtt)
[![Release](https://img.shields.io/github/v/release/ego-component/eemqtt.svg?style=flat-square)](https://github.com/ego-component/eemqtt)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Example](https://img.shields.io/badge/Examples-2ca5e0?style=flat&logo=appveyor)](https://github.com/ego-component/eemqtt/examples/user_auth)
[![Doc](https://img.shields.io/badge/Docs-1?style=flat&logo=appveyor)]()

## Table of contents
- [基本组件](#基本组件)
	- [快速上手](#快速上手)
- [使用说明](#使用说明)
    - [注意事项](#注意事项)
- [测试](#测试)

## 基本组件

对 [eclipse/paho.golang](https://github.com/eclipse/paho.golang) 进行了轻量封装，并提供了以下功能：

- 规范了标准配置格式，提供了统一的 Load().Build() 方法。
- 开启 Debug 后可输出 调试信息至终端。
- 提供了监控拦截器，连接失败计数。
- paho.golang 包 默认支持 MQTT 3.11/5.0 协议，支持断开重连。




### 快速上手

使用样例可参考 [example](examples/user_auth/main.go)

### 注意事项
官网指定 mqtt SDK[paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang) 本组件是另一个分支，支持 5.0协议，但并没有完全支持，作者准备自己完善   

## 使用说明
通过连接到 emqtt[服务端](https://github.com/emqx/emqx) 实现发送主题消息 / 订阅主题消息

配置文件说明
```toml
[emqtt]
debug = true #启用框架自动调试日志
brokers = ["tcp://127.0.0.1:1883"]
#关闭匿名认证后，需要设置用户名和密码才能连接 EMQX_ALLOW_ANONYMOUS=false 
username = ""
password = ""

#通过 tls 安全认证 true 为启用
EnableTLS = false

#如果未设置 客户端正式 只需要配置 ca 文件即可
TLSClientCA = "./certs/cacert.pem"

#订阅的主题配置， 本次配置了两个主题，连接到服务器后将自定订阅这两个主题
[emqtt.subscribeTopics.s1]
topic = "topic1"
qos = 0

[emqtt.subscribeTopics.s2]
topic = "topic2"
qos = 0
```

代码说明
```go
  ...
    //统一的初始化组件方式
	emqClient = eemqtt.Load("emqtt").Build()
    
    //start 组件自动连接服务器，并订阅配置主题     
	emqClient.Start(msgHandler)
     
	//给主题 topic1 发送消息
    emqClient.PublishMsg("topic1", 0, msg)


	//msgHandler 收到订阅主题的消息处理函数
    func msgHandler(ctx context.Context, pp *paho.Publish) {
       elog.Info("receive meg", elog.Any("topic", pp.Topic), elog.Any("msg", string(pp.Payload)))
       //todo 做相关的业务处理
    }

  ...
```
### 注意事项
发送给主题的消息，一定是string 或 []byte 类型    
对象需要转换一下      
```go
//转换后发送 
bytes, _ := json.Marshal(message) 
emqClient.PublishMsg("topic1", 0, bytes) 
```
关于emqtt 相关的概念 例如qos..等等不了解的请自行去 [emqx官网](https://www.emqx.io/docs/zh/v4.4/#emqx-%E6%B6%88%E6%81%AF%E6%9C%8D%E5%8A%A1%E5%99%A8%E5%8A%9F%E8%83%BD%E5%88%97%E8%A1%A8) 学习


## 测试

### 在线测试服务
emqx 官网提供的测试服务[免费的在线 MQTT 5 服务器](https://www.emqx.com/zh/mqtt/public-mqtt5-broker)     
请参考案例 [example](examples/tls_auth/main.go)   

配置信息
```toml
[emqtt]
debug = true #启用框架自动调试日志

#通过官网服务进行测试 https://www.emqx.com/zh/mqtt/public-mqtt5-broker
brokers = ["mqtts://broker-cn.emqx.io:8883"]

#启用证书
EnableTLS = true
TLSClientCA = "./certs/broker.emqx.io-ca.crt"



[emqtt.subscribeTopics.s1]
topic = "topic1"
qos = 1

[emqtt.subscribeTopics.s2]
topic = "topic2"
qos = 1
```


### 自己搭建测试服务
为了方便大家测试，这里提供了一个搭建容器测试环境的案例

#### 1. 安装redis 
requirepass 设置 redis 初始化访问密码 案例中设置为 root
```shell script
docker run -d --name redis -p 6379:6379 redis:latest redis-server --requirepass "root"
```
#### 2. 安装 emqx 服务： 默认启用密码验证 redis 插件
EMQX_AUTH__REDIS__SERVER 是 redis的服务器和端口，填写宿主机真实IP   
EMQX_AUTH__REDIS__PASSWORD redis 访问密码   
EMQX_AUTH__REDIS__PASSWORD_HASH=plain 为了方便测试 密码配置成明文形式   
EMQX_ALLOW_ANONYMOUS = false  默认关闭了 匿名认真，必须输入 用户名和密码才能登录      

```shell script  
docker run -d --name emqx -p 18083:18083 -p 1883:1883 -p 4369:4369 -p 8083:8083 -p 8084:8084  \
    -e EMQX_LISTENER__TCP__EXTERNAL=1883 \
    -e EMQX_LOADED_PLUGINS="emqx_auth_redis,emqx_recon,emqx_retainer,emqx_management,emqx_dashboard" \
    -e EMQX_AUTH__REDIS__SERVER="192.168.1.100:6379" \
    -e EMQX_AUTH__REDIS__PASSWORD="root" \
    -e EMQX_AUTH__REDIS__PASSWORD_HASH=plain \
    -e EMQX_ALLOW_ANONYMOUS="false" \
    emqx/emqx:v4.0.0
```

#### 3. 设置emqx 访问密码
连接上redis命令行，分别执行下面的两条语句设置用户名和密码。     
用户名换成你的用户名，密码换成你的密码。      
```shell script   
  HSET mqtt_user:用户名 is_superuser 1
  HSET mqtt_user:用户名 password 密码
```
#### 4.go do it






