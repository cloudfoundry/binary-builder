package php_test

import (
	"testing"

	"github.com/cloudfoundry/binary-builder/internal/php"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecipeFor_AllKnownKlasses(t *testing.T) {
	knownKlasses := []string{
		"PeclRecipe",
		"FakePeclRecipe",
		"AmqpPeclRecipe",
		"MaxMindRecipe",
		"HiredisRecipe",
		"LibSodiumRecipe",
		"IonCubeRecipe",
		"LuaRecipe",
		"MemcachedPeclRecipe",
		"OdbcRecipe",
		"PdoOdbcRecipe",
		"SodiumRecipe",
		"OraclePeclRecipe",
		"OraclePdoRecipe",
		"PHPIRedisRecipe",
		"RabbitMQRecipe",
		"RedisPeclRecipe",
		"SnmpRecipe",
		"TidewaysXhprofRecipe",
		"LibRdKafkaRecipe",
		"Gd74FakePeclRecipe",
		"EnchantFakePeclRecipe",
	}

	for _, klass := range knownKlasses {
		t.Run(klass, func(t *testing.T) {
			recipe, err := php.RecipeFor(klass)
			require.NoError(t, err, "RecipeFor(%q) should not error", klass)
			assert.NotNil(t, recipe, "RecipeFor(%q) should return non-nil recipe", klass)
		})
	}
}

func TestRecipeFor_UnknownKlass(t *testing.T) {
	_, err := php.RecipeFor("NonExistentRecipe")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown extension klass")
	assert.Contains(t, err.Error(), "NonExistentRecipe")
}

func TestLoad_AllKlassesInBaseFileAreKnown(t *testing.T) {
	// Verify every klass in the real base extensions file maps to a known recipe.
	set, err := php.Load("8", "4")
	require.NoError(t, err)

	all := append(set.NativeModules, set.Extensions...)
	for _, ext := range all {
		if ext.Klass == "" {
			continue
		}
		_, err := php.RecipeFor(ext.Klass)
		assert.NoError(t, err, "klass %q from base extensions should be known", ext.Klass)
	}
}
