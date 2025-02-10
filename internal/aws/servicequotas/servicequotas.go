package servicequotas

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	aws "github.com/georgealton/rain/internal/aws"
)

func getClient() *servicequotas.Client {
	return servicequotas.NewFromConfig(aws.Config())
}

// Get the value for a service quota
func GetQuota(serviceCode string, quotaCode string) (float64, error) {

	res, err := getClient().GetServiceQuota(context.Background(),
		&servicequotas.GetServiceQuotaInput{
			QuotaCode:   &quotaCode,
			ServiceCode: &serviceCode,
		})
	if err != nil {
		return -1, nil
	}
	return *res.Quota.Value, nil
}
