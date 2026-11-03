// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysConfig is the golang structure for table sys_config.
type SysConfig struct {
	Id          uint64      `json:"id"          orm:"id"           description:"参数ID"`
	ConfigKey   string      `json:"configKey"   orm:"config_key"   description:"参数键(唯一)"`
	ConfigValue string      `json:"configValue" orm:"config_value" description:"参数值"`
	Name        string      `json:"name"        orm:"name"         description:"参数名称"`
	Remark      string      `json:"remark"      orm:"remark"       description:"备注"`
	Status      int         `json:"status"      orm:"status"       description:"状态: 1=启用 0=停用"`
	Builtin     int         `json:"builtin"     orm:"builtin"      description:"是否内置: 1=内置"`
	CreatedBy   uint64      `json:"createdBy"   orm:"created_by"   description:"创建人ID"`
	CreatedAt   *gtime.Time `json:"createdAt"   orm:"created_at"   description:"创建时间"`
	UpdatedAt   *gtime.Time `json:"updatedAt"   orm:"updated_at"   description:"更新时间"`
	DeletedAt   *gtime.Time `json:"deletedAt"   orm:"deleted_at"   description:"删除时间(软删除)"`
}
