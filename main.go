package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {
	http.HandleFunc("/tags/", tags)
	http.ListenAndServe(":9000", nil)
}

func tags(w http.ResponseWriter, r *http.Request) {
	ip := strings.Split(r.RemoteAddr, ":")[0]
	instanceID := r.URL.Path[len("/tags/"):]
	instance, err := getInstance(instanceID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error())
		return
	}

	if *instance.PrivateIpAddress != ip {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, fmt.Sprintf("Only %s can retrieve tags for %s", *instance.PrivateIpAddress, instanceID))
		return
	}

	tags := make(map[string]string)

	for _, tag := range instance.Tags {
		tags[*tag.Key] = *tag.Value
	}

	str, err := json.Marshal(&tags)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(str))
}

func getInstance(instanceID string) (ec2.Instance, error) {
	svc := ec2.New(session.New(), &aws.Config{Region: aws.String("us-east-1")})

	filters := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("instance-id"),
				Values: []*string{
					aws.String(instanceID),
				},
			},
		},
	}

	resp, err := svc.DescribeInstances(filters)
	if err != nil {
		return ec2.Instance{}, err
	}

	if len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
		return ec2.Instance{}, errors.New("Instance not found")
	}

	return *resp.Reservations[0].Instances[0], nil
}
