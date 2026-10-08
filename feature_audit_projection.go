package workbench

import (
	"fmt"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func auditFeatureVersion(version string) bool {
	return version == jointdecision.RecordSharedFeatureVersion || version == jointdecision.RecordOriginSharedFeatureVersion ||
		version == jointdecision.RecordGraphSharedFeatureVersion
}

// Each source representation is a separate contract even though all project
// to 768 floats. A row cannot silently supply several forms.
func auditChoiceShape(row FeatureAuditCase, version string) bool {
	switch version {
	case jointdecision.RecordSharedFeatureVersion:
		return len(row.Choices) == 3 && len(row.OriginChoices) == 0 && row.Graph == nil
	case jointdecision.RecordOriginSharedFeatureVersion:
		return len(row.OriginChoices) == 3 && len(row.Choices) == 0 && row.Graph == nil
	case jointdecision.RecordGraphSharedFeatureVersion:
		return row.Graph != nil && len(row.Choices) == 0 && len(row.OriginChoices) == 0
	default:
		return false
	}
}

func projectAuditRow(row FeatureAuditCase, version string, output *[jointdecision.ThreeFeatureDim]float32) error {
	if !auditChoiceShape(row, version) {
		return fmt.Errorf("exactly three choices matching the declared feature contract are required")
	}
	if version == jointdecision.RecordGraphSharedFeatureVersion {
		text, err := jointdecision.EncodeRecordGraphThree(*row.Graph)
		if err != nil {
			return err
		}
		return jointdecision.FeaturesIntoRecordGraphThree(text, output)
	}
	if version == jointdecision.RecordOriginSharedFeatureVersion {
		text, err := jointdecision.EncodeRecordOriginThree([3]jointdecision.RecordOriginChoice(row.OriginChoices))
		if err != nil {
			return err
		}
		return jointdecision.FeaturesIntoRecordOriginThree(text, output)
	}
	text, err := jointdecision.EncodeRecordThree([3]jointdecision.RecordChoice(row.Choices))
	if err != nil {
		return err
	}
	return jointdecision.FeaturesIntoRecordThree(text, output)
}

func predictAuditFeatures(model *jointdecision.ThreeModel, features *[jointdecision.ThreeFeatureDim]float32,
	workspace *jointdecision.ThreeWorkspace, output *jointdecision.ThreePrediction) error {
	if model.FeatureVersion() == jointdecision.RecordGraphSharedFeatureVersion {
		return model.PredictRecordGraphSharedFeaturesInto(features, workspace, output)
	}
	if model.FeatureVersion() == jointdecision.RecordOriginSharedFeatureVersion {
		return model.PredictRecordOriginSharedFeaturesInto(features, workspace, output)
	}
	return model.PredictRecordSharedFeaturesInto(features, workspace, output)
}

func loadAuditModel(path, version string) (*jointdecision.ThreeModel, error) {
	switch version {
	case jointdecision.RecordSharedFeatureVersion:
		return jointdecision.LoadRecordSharedThree(path)
	case jointdecision.RecordOriginSharedFeatureVersion:
		return jointdecision.LoadRecordOriginSharedThree(path)
	case jointdecision.RecordGraphSharedFeatureVersion:
		return jointdecision.LoadRecordGraphSharedThree(path)
	default:
		return nil, fmt.Errorf("unsupported audit model feature contract")
	}
}
