package manager

import (
    "encoding/json"
    "fmt"
    "net"
    "os"
    "sync"
    "time"

    "github.com/aws/aws-sdk-go/aws"
    awssess "github.com/aws/aws-sdk-go/aws/session"
    awsec2 "github.com/aws/aws-sdk-go/service/ec2"
)

// EC2
type EC2Manager struct {
    CredentialsPath string `json:"credentialsPath"`
    InstanceId string `json:"instanceId"`
    Region string `json:"region"`
    Profile string `json:"profile"`
    Port uint16 `json:"port"`
    Timeout int `json:"timeout"` // unit: seconds
    publicDnsName string
    appState int
    appTime time.Time
    lock sync.Mutex
}

func newEC2Manager() *EC2Manager {
    return &EC2Manager{
        publicDnsName: "",
        appState: StateObscure,
    }
}

func NewEC2Manager(cp, id, rg, pf string, p uint16, to int) *EC2Manager {
    ec2 := newEC2Manager()
    ec2.CredentialsPath = cp
    ec2.InstanceId = id
    ec2.Region = rg
    ec2.Profile = pf
    ec2.Port = p
    ec2.Timeout = to
    return ec2
}

func NewEC2ManagerJson(data []byte) (*EC2Manager, error) {
    ec2 := newEC2Manager()
    return ec2, json.Unmarshal(data, ec2)
}

func(ec2 *EC2Manager) newService() (*awsec2.EC2, error) {

    err := os.Setenv("AWS_SHARED_CREDENTIALS_FILE", ec2.CredentialsPath)
    if err != nil {
        return nil, err
    }

    sess, err := awssess.NewSessionWithOptions(awssess.Options{
        Profile: ec2.Profile,
        Config: aws.Config{
            Region: aws.String(ec2.Region),
        },
    })
    if err != nil {
        return nil, err
    }

    return awsec2.New(sess), nil

}

func(ec2 *EC2Manager) instanceIds() []*string {
    return []*string{aws.String(ec2.InstanceId)}
}

func(ec2 *EC2Manager) Addr() string {
    portstr := fmt.Sprintf("%d", ec2.Port)
    return net.JoinHostPort(ec2.publicDnsName, portstr)
}

func(ec2 *EC2Manager) Start() error {

    ec2.lock.Lock()
    defer ec2.lock.Unlock()

    svc, err := ec2.newService()
    if err != nil {
        return err
    }

    input := &awsec2.StartInstancesInput{
        InstanceIds: ec2.instanceIds(),
    }
    _, err = svc.StartInstances(input)
    if err != nil {
        return err
    }

    // Start app state watcher
    go func() {

        ec2.appState = StateStopped

        for {
            after := time.After(ec2.timeout())

            conn, err := ec2.Dial()
            ec2.appTime = time.Now()

            if err == nil { // Connected
                conn.Close()
                ec2.appState = StateRunning
                return
            }

            ec2.appState = StatePending
            <-after
        }

    }()

    return nil

}

func(ec2 *EC2Manager) State() (int, error) {

    ec2.lock.Lock()
    defer ec2.lock.Unlock()

    svc, err := ec2.newService()
    if err != nil {
        return StateObscure, err
    }

    input := &awsec2.DescribeInstancesInput{
        InstanceIds: ec2.instanceIds(),
    }
    result, err := svc.DescribeInstances(input)
    if err != nil {
        return StateObscure, err
    }

    instance := result.Reservations[0].Instances[0]
    // Dns and code
    switch *instance.State.Code {
    case 0: // EC2 pending
        return StatePending, nil
    case 16: // EC2 running
        ec2.publicDnsName = *instance.PublicDnsName
        // Check underlying server
        start := time.Now()
        conn, err := ec2.Dial()
        if err != nil {
            switch ec2.appState {
            case StateStopped, StatePending: // Currently pending
                return StatePending, nil
            case StateRunning:
                if start.Before(ec2.appTime) {
                    return StateRunning, nil
                }
                ec2.appState = StateStopping
                return StateStopping, nil
            case StateStopping:
                return StateStopping, nil
            }
            // State obscure and others
            return StateObscure, nil
        }
        conn.Close()

        return StateRunning, nil
    case 64: // EC2 stopping
        return StateStopping, nil
    case 80: // EC2 stopped
        return StateStopped, nil
    }

    return StateObscure, nil

}

func(ec2 *EC2Manager) timeout() time.Duration {
    return time.Duration(ec2.Timeout) * time.Second
}

func(ec2 *EC2Manager) Dial() (net.Conn, error) {
    return dialTimeout(ec2.Addr(), ec2.timeout())
}
