package service

import "errors"

var (
	ErrInvalidInput       = errors.New("请求参数不合法")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserAlreadyExists  = errors.New("用户名或邮箱已存在")
	ErrPaperNotFound      = errors.New("论文不存在")
	ErrFileAlreadyExists  = errors.New("文件已存在，请勿重复上传")
)
