package php

import (
	"context"
	"fmt"

	"github.com/cloudfoundry/binary-builder/internal/fetch"
	"github.com/cloudfoundry/binary-builder/internal/runner"
)

// ExtensionContext holds the runtime paths resolved during the PHP build
// and passed to each extension recipe.
type ExtensionContext struct {
	PHPPath       string // e.g. /app/vendor/php-8.3.x
	PHPSourceDir  string // e.g. /tmp/php-8.3.x (unpacked PHP source, used by FakePecl)
	HiredisPath   string // set after hiredis native module builds
	LibSodiumPath string // set after libsodium native module builds
	LuaPath       string // set after lua native module builds
	RabbitMQPath  string // set after rabbitmq native module builds
	IonCubePath   string // set after ioncube download
	PHPMajor      string // e.g. "8"
	PHPMinor      string // e.g. "3"
	Fetcher       fetch.Fetcher
}

// ExtensionRecipe is the interface implemented by every PHP extension builder.
type ExtensionRecipe interface {
	// Build performs the full build cycle for the given extension.
	Build(ctx context.Context, ext Extension, ec ExtensionContext, run runner.Runner) error
}

// RecipeFor returns the ExtensionRecipe implementation for the given klass name.
// Returns an error for unknown klass names.
func RecipeFor(klass string) (ExtensionRecipe, error) {
	switch klass {
	case "PeclRecipe":
		return &PeclRecipe{}, nil
	case "FakePeclRecipe":
		return &FakePeclRecipe{}, nil
	case "AmqpPeclRecipe":
		return &AmqpPeclRecipe{}, nil
	case "MaxMindRecipe":
		return &MaxMindRecipe{}, nil
	case "HiredisRecipe":
		return &HiredisRecipe{}, nil
	case "LibSodiumRecipe":
		return &LibSodiumRecipe{}, nil
	case "IonCubeRecipe":
		return &IonCubeRecipe{}, nil
	case "LuaRecipe":
		return &LuaRecipe{}, nil
	case "MemcachedPeclRecipe":
		return &MemcachedPeclRecipe{}, nil
	case "OdbcRecipe":
		return &OdbcRecipe{}, nil
	case "PdoOdbcRecipe":
		return &PdoOdbcRecipe{}, nil
	case "SodiumRecipe":
		return &SodiumRecipe{}, nil
	case "OraclePeclRecipe":
		return &OraclePeclRecipe{}, nil
	case "OraclePdoRecipe":
		return &OraclePdoRecipe{}, nil
	case "PHPIRedisRecipe":
		return &PHPIRedisRecipe{}, nil
	case "RabbitMQRecipe":
		return &RabbitMQRecipe{}, nil
	case "RedisPeclRecipe":
		return &RedisPeclRecipe{}, nil
	case "SnmpRecipe":
		return &SnmpRecipe{}, nil
	case "TidewaysXhprofRecipe":
		return &TidewaysXhprofRecipe{}, nil
	case "LibRdKafkaRecipe":
		return &LibRdKafkaRecipe{}, nil
	case "Gd74FakePeclRecipe":
		return &Gd74FakePeclRecipe{}, nil
	case "EnchantFakePeclRecipe":
		return &EnchantFakePeclRecipe{}, nil
	default:
		return nil, fmt.Errorf("php: unknown extension klass %q", klass)
	}
}
