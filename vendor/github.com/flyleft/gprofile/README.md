# [gprofile](https://github.com/flyleft/gprofile)
> 一个简单的go语言配置文件库，将YAML配置自动映射到实体中, 类似与spring-boot配置文件。

- 根实体strut遍历实体属性, 如果属性为strut则递归遍历。
- 根据tag中的`profile`值(不设置则为当前属性名的首字母变小写)解析YAML或flag或环境变量，并赋值, 可通过`profile:"_"`跳过该属性.
- 若YAML或flag或环境变量都不存在，则使用tag中的`profileDefault`设置默认值。
- 默认变量优先级：环境变量 > flag参数 > YAML > profileDefault默认值
- 支持类型: string、bool、uint、uint8、uint16、Uint32、Uint64、int、int8、int16、int32、int64、float32、float64、
[]string、[]bool、[]uint、[]uint8、[]uint16、[]Uint32、[]Uint64、[]int、[]int8、[]int16、[]int32、[]int64、[]float32、[]float64、map[string]interface{}

注意：**接收类型和返回类型都需要为指针类型**。

### 单profile下的使用：
```yml
eureka:
  instance:
    preferIpAddress: true
    leaseRenewalIntervalInSeconds: 10
    leaseExpirationDurationInSeconds: 30
  client:
    serviceUrl:
      defaultZone: http://localhost:8000/eureka/
    registryFetchIntervalSeconds: 10
logging:
  level:
    github.com/flyleft/consul-iris/pkg/config: info
    github.com/flyleft/consul-iris/pkg/route: debug
```

```go
type SingleEnv struct {
    Skip       string               `profile:"_"` //跳过设置该属性
	Eureka  SingleEureka
	Logging map[string]interface{} `profile:"logging.level" profileDefault:"{\"github.com/flyleft/consul-iris\":\"debug\"}"`
}

type SingleEureka struct {
	PreferIpAddress                  bool   `profile:"instance.preferIpAddress"`
	LeaseRenewalIntervalInSeconds    int32  `profile:"instance.leaseRenewalIntervalInSeconds"`
	LeaseExpirationDurationInSeconds uint   `profile:"instance.leaseExpirationDurationInSeconds"`
	ServerDefaultZone                string `profile:"client.serviceUrl.defaultZone" profileDefault:"http://localhost:8000/eureka/"`
	RegistryFetchIntervalSeconds     byte   `profile:"client.registryFetchIntervalSeconds"`
}

//使用
func main()  {
	env, err := Profile(&SingleEnv{}, "test-single-profile.yml", true)
	if err != nil {
		t.Error("Profile execute error", err)
	}
	trueEnv := env.(*SingleEnv)
}
//通过环境变量覆盖配置，比如设置EUREKA_INSTANCE_LEASERENEWALINTERVALINSECONDS环境变量值覆盖eureka.instance.leaseRenewalIntervalInSeconds
```


### 多profile下的使用：
```yml
profiles:
  active: dev # 设置生效的profile

dev:
  database:
    username: root
    password: root
  eureka:
    zone: http://localhost:8000/eureka/
    fetchInterval: 10
  logging:
    level:
      github.com/flyleft/consul-iris/pkg/config: info
      github.com/flyleft/consul-iris/pkg/route: debug


production:
  database:
    username: production
    password: production
  eureka:
    zone: http://localhost:8000/eureka/
    fetchInterval: 10
  logging:
    level:
      github.com/flyleft/consul-iris/pkg/config: info
      github.com/flyleft/consul-iris/pkg/route: debug

```

```go
type Eureka struct {
	Zone          string `profile:"zone"`
	FetchInterval int    `profile:"fetchInterval"`
}

type DataSource struct {
	Host     string `profile:"host" profileDefault:"localhost"`
	Username string `profile:"username"`
	Password string `profile:"password"`
}

type MultiEnv struct {
	DataSource DataSource `profile:"database"`
	Eureka     Eureka
	Logging    map[string]interface{} `profile:"logging.level" profileDefault:"{\"github.com/flyleft/consul-iris\":\"debug\"}"`
	Users      []interface{}          `profile:"users" profileDefault:"[\"admin\",\"test\",\"root\"]"`
}
func main()  {
	env, err := Profile(&MultiEnv{}, "test-multi-profile.yml", true)
	if err != nil {
		t.Error("Profile execute error", err)
	}
	trueEnv := env.(*MultiEnv)
}
//可通过环境变量DEV_EUREKA_ZONE覆盖eureka.zone的值
```