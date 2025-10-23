# Agent Guidelines for Binary Builder

## Test Commands
- Run all tests: `bundle exec rspec`
- Run single test: `bundle exec rspec spec/integration/ruby_spec.rb`
- Exclude Oracle PHP tests: `bundle exec rspec --tag ~run_oracle_php_tests`

## Lint Commands
- Run RuboCop: `bundle exec rubocop`

## Code Style
- **Encoding**: Add `# encoding: utf-8` at the top of all Ruby files
- **Imports**: Use `require_relative` for local files, `require` for gems
- **Naming**: Use snake_case for methods/variables, CamelCase for classes
- **Classes**: Recipe classes inherit from `BaseRecipe` or `MiniPortile`
- **Error Handling**: Use `or raise 'Error message'` for critical failures, check `$?.success?` for command execution
- **String Interpolation**: Prefer double quotes and `#{}` for interpolation
- **Methods**: Define helper methods as private when appropriate

## Recipe Patterns
- Override `computed_options`, `url`, `archive_files`, `prefix_path` in recipe classes
- Use `execute()` for build steps, `run()` for apt/system commands
- Place recipes in `recipe/` directory, tests in `spec/integration/`
