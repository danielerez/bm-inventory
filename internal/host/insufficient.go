package host

import (
	"context"

	"github.com/filanov/bm-inventory/internal/hardware"
	"github.com/filanov/bm-inventory/models"
	logutil "github.com/filanov/bm-inventory/pkg/log"
	"github.com/go-openapi/swag"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func NewInsufficientState(log logrus.FieldLogger, db *gorm.DB, hwValidator hardware.Validator) *insufficientState {
	return &insufficientState{
		baseState: baseState{
			log: log,
			db:  db,
		},
		hwValidator: hwValidator,
	}
}

type insufficientState struct {
	baseState
	hwValidator hardware.Validator
}

func (i *insufficientState) UpdateHwInfo(ctx context.Context, h *models.Host, hwInfo string) (*UpdateReply, error) {
	h.HardwareInfo = hwInfo
	return updateHwInfo(logutil.FromContext(ctx, i.log), i.hwValidator, h, i.db)
}

func (d *insufficientState) UpdateInventory(ctx context.Context, h *models.Host, inventory string) (*UpdateReply, error) {
	h.Inventory = inventory
	return updateStateFromInventory(logutil.FromContext(ctx, d.log), d.hwValidator, h, d.db)
}

func (i *insufficientState) UpdateRole(ctx context.Context, h *models.Host, role string, db *gorm.DB) (*UpdateReply, error) {
	log := logutil.FromContext(ctx, i.log)
	cdb := i.db
	if db != nil {
		cdb = db
	}
	cluster, err := getCluster(h.ClusterID, cdb)
	if err != nil {
		return nil, err
	}
	reply, err := i.hwValidator.IsSufficient(h, cluster)
	if err != nil {
		return nil, err
	}
	if !reply.IsSufficient {
		return updateStateWithParams(log, HostStatusInsufficient, reply.Reason, h, cdb,
			"role", role)
	}
	return updateStateWithParams(log, HostStatusKnown, "", h, cdb, "role", role)
}

func (i *insufficientState) RefreshStatus(ctx context.Context, h *models.Host, db *gorm.DB) (*UpdateReply, error) {
	if db == nil {
		db = i.db
	}
	reply, err := updateByKeepAlive(logutil.FromContext(ctx, i.log), h, db)
	if err != nil || reply.IsChanged || h.Inventory == "" {
		return reply, err
	}
	return updateStateFromInventory(logutil.FromContext(ctx, i.log), i.hwValidator, h, db)
}

func (i *insufficientState) Install(ctx context.Context, h *models.Host, db *gorm.DB) (*UpdateReply, error) {
	return nil, errors.Errorf("unable to install host <%s> in <%s> status",
		h.ID, swag.StringValue(h.Status))
}

func (i *insufficientState) EnableHost(ctx context.Context, h *models.Host) (*UpdateReply, error) {
	// State in the same state
	return &UpdateReply{
		State:     HostStatusInsufficient,
		IsChanged: false,
	}, nil
}

func (i *insufficientState) DisableHost(ctx context.Context, h *models.Host) (*UpdateReply, error) {
	return updateState(logutil.FromContext(ctx, i.log), HostStatusDisabled, statusInfoDisabled, h, i.db)
}
