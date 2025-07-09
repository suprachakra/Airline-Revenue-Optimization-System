package forecasting

import (
	"context"
	"log"
	"iaros/forecasting_service/src/model"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sagemaker"
)

// ModelTrainer automates the retraining of forecasting models using AWS SageMaker.
func ModelTrainer(ctx context.Context, data []float64, modelName string) error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}
	svc := sagemaker.New(sess)

	inputConfig := model.GetSageMakerInputConfig(data)
	outputConfig := model.GetSageMakerOutputConfig(modelName)

	_, err = svc.CreateTrainingJob(&sagemaker.CreateTrainingJobInput{
		TrainingJobName:   &modelName,
		InputDataConfig:   inputConfig,
		OutputDataConfig:  outputConfig,
		ResourceConfig:    model.GetResourceConfig(),
		RoleArn:           model.GetSageMakerRoleArn(),
	})
	if err != nil {
		log.Printf("ModelTrainer failed for %s: %v", modelName, err)
		return err
	}
	log.Printf("Retraining job for %s initiated successfully", modelName)
	return nil
}
