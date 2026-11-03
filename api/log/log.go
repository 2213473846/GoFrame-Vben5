// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package log

import (
	"context"

	"demo/api/log/v1"
)

type ILogV1 interface {
	LoginLogList(ctx context.Context, req *v1.LoginLogListReq) (res *v1.LoginLogListRes, err error)
	OperationLogList(ctx context.Context, req *v1.OperationLogListReq) (res *v1.OperationLogListRes, err error)
	LoginLogExport(ctx context.Context, req *v1.LoginLogExportReq) (res *v1.LoginLogExportRes, err error)
	OperationLogExport(ctx context.Context, req *v1.OperationLogExportReq) (res *v1.OperationLogExportRes, err error)
}
