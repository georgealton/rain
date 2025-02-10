package forecast

import (
	"github.com/georgealton/rain/internal/aws/kms"
	"github.com/georgealton/rain/internal/console/spinner"
	fc "github.com/georgealton/rain/plugins/forecast"
)

func CheckSNSTopic(input fc.PredictionInput) fc.Forecast {

	forecast := fc.MakeForecast(&input)

	spin(input.TypeName, input.LogicalId, "Checking SNS Topic Key")
	checkSNSTopicKey(&input, &forecast)
	spinner.Pop()

	return forecast
}

func checkSNSTopicKey(input *fc.PredictionInput, forecast *fc.Forecast) {

	// Get the KmsMasterKeyId from the input resource properties
	k := input.GetPropertyNode("KmsMasterKeyId")
	if k != nil {
		keyArn := k.Value
		valid := kms.IsKeyArnValid(keyArn)
		if valid {
			forecast.Add(F0013, true, "KMS Key is valid", getLineNum(input.LogicalId, input.Resource))
		} else {
			forecast.Add(F0013, false, "KMS Key is invalid", getLineNum(input.LogicalId, input.Resource))
		}
	}
}
