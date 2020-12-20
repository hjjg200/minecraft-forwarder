package manager

import (
    "fmt"
    "net"
    "os"
    "sync"
    "time"

    "github.com/aws/aws-sdk-go/aws"
    awssess "github.com/aws/aws-sdk-go/aws/session"
    awsec2 "github.com/aws/aws-sdk-go/service/ec2"
)

const (
    EC2AppWatchInterval = time.Second * 5
)

// EC2
type EC2Manager struct {
    CredentialsPath string `json:"credentialsPath"`
    InstanceID string `json:"instanceId"`
    Region string `json:"region"`
    Profile string `json:"profile"`
    Port uint16 `json:"port"`
    Timeout int `json:"timeout"` // unit: seconds
    publicDnsName string
    appState int
    lock sync.Mutex
}

func NewEC2Manager(cp, id, rg, pf string, p uint16, to int) *EC2Manager {
    return &EC2Manager{
        CredentialsPath: cp,
        InstanceID: id,
        Region: rg,
        Profile: pf,
        Port: p,
        Timeout: to,
    }
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
    return []*string{aws.String(ec2.InstanceID)}
}

func(ec2 *EC2Manager) addr() string {
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
            conn, err := ec2.dial()

            if err != nil {
                ec2.appState = StatePending
            } else {
                conn.Close()
                ec2.appState = StateRunning
                return
            }

            time.Sleep(EC2AppWatchInterval)
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
        conn, err := ec2.dial()
        if err != nil {
            switch ec2.appState {
            case StateStopped, StatePending: // Currently pending
                return StatePending, nil
            case StateRunning: // Previously running
                ec2.appState = StateStopping
                return StateStopping, nil
            case StateStopping:
                return StateStopping, nil
            }
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

func(ec2 *EC2Manager) dial() (net.Conn, error) {
    return dialTimeout(ec2.addr(), time.Duration(ec2.Timeout) * time.Second)
}

func(ec2 *EC2Manager) Dial() (net.Conn, error) {

    state, err := ec2.State()
    if err != nil {
        return nil, err
    }
    if state != StateRunning {
        return nil, fmt.Errorf("Server is not running")
    }

    return ec2.dial()

}
