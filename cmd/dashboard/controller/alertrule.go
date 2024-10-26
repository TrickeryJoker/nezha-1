package controller

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
)

// List Alert rules
// @Summary List Alert rules
// @Security BearerAuth
// @Schemes
// @Description List Alert rules
// @Tags auth required
// @Produce json
// @Success 200 {object} model.CommonResponse[[]model.AlertRule]
// @Router /alert-rule [get]
func listAlertRule(c *gin.Context) ([]*model.AlertRule, error) {
	singleton.AlertsLock.RLock()
	defer singleton.AlertsLock.RUnlock()

	var ar []*model.AlertRule
	if err := copier.Copy(&ar, &singleton.Alerts); err != nil {
		return nil, err
	}
	return ar, nil
}

// Add Alert Rule
// @Summary Add Alert Rule
// @Security BearerAuth
// @Schemes
// @Description Add Alert Rule
// @Tags auth required
// @Accept json
// @param request body model.AlertRuleForm true "AlertRuleForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[uint64]
// @Router /alert-rule [post]
func createAlertRule(c *gin.Context) (uint64, error) {
	var arf model.AlertRuleForm
	var r model.AlertRule

	if err := c.ShouldBindJSON(&arf); err != nil {
		return 0, err
	}

	if err := validateRule(&r); err != nil {
		return 0, err
	}

	r.Name = arf.Name
	r.Rules = arf.Rules
	r.FailTriggerTasks = arf.FailTriggerTasks
	r.RecoverTriggerTasks = arf.RecoverTriggerTasks
	r.NotificationGroupID = arf.NotificationGroupID
	enable := arf.Enable
	r.TriggerMode = arf.TriggerMode
	r.Enable = &enable
	r.ID = arf.ID

	if err := singleton.DB.Create(&r).Error; err != nil {
		return 0, newGormError("%v", err)
	}

	singleton.OnRefreshOrAddAlert(r)
	return r.ID, nil
}

// Update Alert Rule
// @Summary Update Alert Rule
// @Security BearerAuth
// @Schemes
// @Description Update Alert Rule
// @Tags auth required
// @Accept json
// @param id path uint true "Alert ID"
// @param request body model.AlertRuleForm true "AlertRuleForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /alert-rule/{id} [patch]
func updateAlertRule(c *gin.Context) (any, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return nil, err
	}

	var arf model.AlertRuleForm
	if err := c.ShouldBindJSON(&arf); err != nil {
		return 0, err
	}

	var r model.AlertRule
	if err := singleton.DB.First(&r, id).Error; err != nil {
		return nil, fmt.Errorf("alert id %d does not exist", id)
	}

	if err := validateRule(&r); err != nil {
		return 0, err
	}

	r.Name = arf.Name
	r.Rules = arf.Rules
	r.FailTriggerTasks = arf.FailTriggerTasks
	r.RecoverTriggerTasks = arf.RecoverTriggerTasks
	r.NotificationGroupID = arf.NotificationGroupID
	enable := arf.Enable
	r.TriggerMode = arf.TriggerMode
	r.Enable = &enable
	r.ID = arf.ID

	if err := singleton.DB.Save(&r).Error; err != nil {
		return 0, newGormError("%v", err)
	}

	singleton.OnRefreshOrAddAlert(r)
	return r.ID, nil
}

// Batch delete Alert rules
// @Summary Batch delete Alert rules
// @Security BearerAuth
// @Schemes
// @Description Batch delete Alert rules
// @Tags auth required
// @Accept json
// @param request body []uint64 true "id list"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /batch-delete/alert-rule [post]
func batchDeleteAlertRule(c *gin.Context) (any, error) {
	var ar []uint64

	if err := c.ShouldBindJSON(&ar); err != nil {
		return nil, err
	}

	if err := singleton.DB.Unscoped().Delete(&model.DDNSProfile{}, "id in (?)", ar).Error; err != nil {
		return nil, newGormError("%v", err)
	}

	singleton.OnDeleteAlert(ar)
	return nil, nil
}

func validateRule(r *model.AlertRule) error {
	if len(r.Rules) > 0 {
		for _, rule := range r.Rules {
			if !rule.IsTransferDurationRule() {
				if rule.Duration < 3 {
					return errors.New("错误: Duration 至少为 3")
				}
			} else {
				if rule.CycleInterval < 1 {
					return errors.New("错误: cycle_interval 至少为 1")
				}
				if rule.CycleStart == nil {
					return errors.New("错误: cycle_start 未设置")
				}
				if rule.CycleStart.After(time.Now()) {
					return errors.New("错误: cycle_start 是个未来值")
				}
			}
		}
	} else {
		return errors.New("至少定义一条规则")
	}
	return nil
}