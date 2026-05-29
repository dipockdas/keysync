/// Example: Using keysync Dart client library
///
/// This example demonstrates retrieving secrets at different scopes:
/// - Global secrets (available to all projects)
/// - Project-scoped secrets (specific to one project)
/// - Environment-scoped secrets (project + environment)

import 'package:keysync/keysync.dart';

Future<void> main() async {
  print('Keysync Dart Client Example\n');

  // Example 1: Global secret
  // Retrieves from environment variable or global keychain scope
  try {
    final apiKey = await getSecret('API_KEY');
    print('Global API Key: $apiKey');
  } catch (e) {
    print('Global API Key not found: $e');
  }

  print('');

  // Example 2: Project-scoped secret
  // Retrieves from environment variable or project-scoped keychain
  try {
    final dbUrl = await getSecret('DATABASE_URL', project: 'myapp');
    print('Project Database URL: $dbUrl');
  } catch (e) {
    print('Project Database URL not found: $e');
  }

  print('');

  // Example 3: Environment-scoped secret
  // Retrieves from environment variable or environment-scoped keychain
  try {
    final prodDbUrl = await getSecret(
      'DATABASE_URL',
      project: 'myapp',
      environment: 'production',
    );
    print('Production Database URL: $prodDbUrl');
  } catch (e) {
    print('Production Database URL not found: $e');
  }

  print('');

  // Example 4: Multiple secrets with error handling
  final Map<String, String?> secrets = {
    'STRIPE_KEY': null,
    'SENDGRID_API_KEY': null,
    'JWT_SECRET': null,
  };

  for (final key in secrets.keys) {
    try {
      secrets[key] = await getSecret(key, project: 'myapp');
      print('Retrieved $key successfully');
    } catch (e) {
      print('Failed to retrieve $key: $e');
    }
  }

  print('');
  print('Example complete!');
}
