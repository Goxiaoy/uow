package uow

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"strings"
)

type Manager interface {
	// WithNew create a new unit of work and execute [fn] with this unit of work
	WithNew(ctx context.Context, fn func(ctx context.Context) error, opt ...*sql.TxOptions) error
}

type KeyFormatter func(keys ...string) string

var (
	DefaultKeyFormatter KeyFormatter = func(keys ...string) string {
		return strings.Join(keys, "/")
	}
)

type IdGenerator func(ctx context.Context) string

var (
	DefaultIdGenerator IdGenerator = func(ctx context.Context) string {
		return uuid.New().String()
	}
)

type manager struct {
	cfg     *Config
	factory DbFactory
}

type Config struct {
	NestedTransaction bool
	formatter         KeyFormatter
	idGen             IdGenerator
}

type Option func(*Config)

func WithNestedNestedTransaction() Option {
	return func(config *Config) {
		config.NestedTransaction = true
	}
}
func WithKeyFormatter(f KeyFormatter) Option {
	return func(config *Config) {
		config.formatter = f
	}
}

func WithIdGenerator(idGen IdGenerator) Option {
	return func(config *Config) {
		config.idGen = idGen
	}
}

func NewManager(factory DbFactory, opts ...Option) Manager {
	cfg := &Config{
		formatter: DefaultKeyFormatter,
		idGen:     DefaultIdGenerator,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &manager{
		cfg:     cfg,
		factory: factory,
	}
}

func (m *manager) WithNew(ctx context.Context, fn func(ctx context.Context) error, opt ...*sql.TxOptions) error {
	factory := m.factory
	//get current for nested
	var parent *unitOfWork
	if m.cfg.NestedTransaction {
		current, ok := FromCurrentUow(ctx)
		if ok {
			parent = current
		}
	}
	if parent != nil {
		//first level uow will use default factory, others will find from parent
		factory = nil
	}
	uow := newUnitOfWork(m.cfg.idGen(ctx), parent, factory, m.cfg.formatter, opt...)
	newCtx := newCurrentUow(ctx, uow)
	return withUnitOfWork(newCtx, fn)
}
