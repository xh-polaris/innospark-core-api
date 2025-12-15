package cache

import (
	"context"
	"time"
)

var Nil error

func SetDefaultNilError(err error) {
	Nil = err
}

type Cmdable interface {
	Pipeline() Pipeliner
	StringCmdable
	HashCmdable
	GenericCmdable
	ListCmdable
	ScriptingCmdable
}

type ScriptingCmdable interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) Cmd
	//EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) Cmd
	//ScriptExists(ctx context.Context, hashes ...string) BoolSliceCmd
	//ScriptLoad(ctx context.Context, script string) StringCmd
}

type StringCmdable interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) StatusCmd
	Get(ctx context.Context, key string) StringCmd
	IncrBy(ctx context.Context, key string, value int64) IntCmd
	Incr(ctx context.Context, key string) IntCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) BoolCmd
}

type HashCmdable interface {
	HSet(ctx context.Context, key string, values ...interface{}) IntCmd
	HGetAll(ctx context.Context, key string) MapStringStringCmd
	HDel(ctx context.Context, key string) IntCmd
}

type GenericCmdable interface {
	Del(ctx context.Context, keys ...string) IntCmd
	Exists(ctx context.Context, keys ...string) IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) BoolCmd
}

type Pipeliner interface {
	StatefulCmdable
	Exec(ctx context.Context) ([]Cmder, error)
}

type StatefulCmdable interface {
	Cmdable
}

type ListCmdable interface {
	LIndex(ctx context.Context, key string, index int64) StringCmd
	LPush(ctx context.Context, key string, values ...interface{}) IntCmd
	RPush(ctx context.Context, key string, values ...interface{}) IntCmd
	LSet(ctx context.Context, key string, index int64, value interface{}) StatusCmd
	LPop(ctx context.Context, key string) StringCmd
	LRange(ctx context.Context, key string, start, stop int64) StringSliceCmd
}
type Cmder interface {
	Err() error
}

type baseCmd interface {
	Err() error
}

type Cmd interface {
	baseCmd
	Result() (interface{}, error)
	Val() interface{}
	Int64() (int64, error)
	String() string
	Bool() (bool, error)
	Slice() ([]interface{}, error)
}

type IntCmd interface {
	baseCmd
	Result() (int64, error)
}

type MapStringStringCmd interface {
	baseCmd
	Result() (map[string]string, error)
}

type BoolCmd interface {
	baseCmd
	Result() (bool, error)
}

type BoolSliceCmd interface {
	baseCmd
	Result() ([]bool, error)
}

type StatusCmd interface {
	baseCmd
	Result() (string, error)
}

type StringCmd interface {
	baseCmd
	Result() (string, error)
	Val() string
	Int64() (int64, error)
	Bytes() ([]byte, error)
}

type StringSliceCmd interface {
	baseCmd
	Result() ([]string, error)
}
