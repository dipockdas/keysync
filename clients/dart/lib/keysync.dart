/// Keysync client library for Dart/Flutter
///
/// Retrieves secrets from environment variables (primary) or OS keychain (fallback).
/// Supports macOS, Linux, and Windows desktop applications.
///
/// Example:
/// ```dart
/// import 'package:keysync/keysync.dart';
///
/// void main() async {
///   // Global scope
///   final apiKey = await getSecret('API_KEY');
///
///   // Project scope
///   final dbUrl = await getSecret('DATABASE_URL', project: 'myapp');
///
///   print('API Key: $apiKey');
///   print('Database URL: $dbUrl');
/// }
/// ```
library keysync;

import 'dart:io';

/// Exception thrown when a secret cannot be retrieved
class KeysyncException implements Exception {
  final String message;
  KeysyncException(this.message);

  @override
  String toString() => 'KeysyncException: $message';
}

/// Retrieves a secret from environment variables or OS keychain.
///
/// Arguments:
/// - [key]: The secret key name (e.g., 'DATABASE_URL')
/// - [project]: Optional project name for project-scoped secrets
/// - [environment]: Optional environment name (e.g., 'dev', 'staging', 'production')
///
/// Returns the secret value as a String.
///
/// Throws [KeysyncException] if the secret is not found or cannot be retrieved.
///
/// Resolution order (highest to lowest precedence):
/// 1. Environment variable matching the key name
/// 2. OS keychain: environment-scoped (if environment provided)
/// 3. OS keychain: project-scoped (if project provided)
/// 4. OS keychain: global scope
Future<String> getSecret(
  String key, {
  String? project,
  String? environment,
}) async {
  // 1. Check environment variables first (primary path)
  final envValue = Platform.environment[key];
  if (envValue != null && envValue.isNotEmpty) {
    return envValue;
  }

  // 2. Fall back to OS keychain with scope resolution
  // Try environment scope first (highest precedence)
  if (project != null && environment != null) {
    try {
      return await _getFromKeychain(key,
          project: project, environment: environment);
    } catch (_) {
      // Fall through to next scope
    }
  }

  // Try project scope
  if (project != null) {
    try {
      return await _getFromKeychain(key, project: project);
    } catch (_) {
      // Fall through to global scope
    }
  }

  // Try global scope
  try {
    return await _getFromKeychain(key);
  } catch (e) {
    throw KeysyncException(
      'Secret "$key" not found in environment variables or OS keychain',
    );
  }
}

/// Internal function to retrieve a secret from the OS keychain
Future<String> _getFromKeychain(
  String key, {
  String? project,
  String? environment,
}) async {
  if (Platform.isMacOS) {
    return await _getMacOS(key, project: project, environment: environment);
  } else if (Platform.isLinux) {
    return await _getLinux(key, project: project, environment: environment);
  } else if (Platform.isWindows) {
    return await _getWindows(key, project: project, environment: environment);
  } else {
    throw KeysyncException(
      'Unsupported platform: ${Platform.operatingSystem}',
    );
  }
}

/// Retrieve secret from macOS Keychain using the security CLI
Future<String> _getMacOS(
  String key, {
  String? project,
  String? environment,
}) async {
  final service = _buildServiceName(project: project, environment: environment);

  final result = await Process.run(
    'security',
    ['find-generic-password', '-s', service, '-a', key, '-w'],
    runInShell: false,
  );

  if (result.exitCode != 0) {
    throw KeysyncException(
      'Failed to retrieve secret from macOS Keychain: ${result.stderr}',
    );
  }

  return (result.stdout as String).trim();
}

/// Retrieve secret from Linux secret-tool (libsecret)
Future<String> _getLinux(
  String key, {
  String? project,
  String? environment,
}) async {
  final service = _buildServiceName(project: project, environment: environment);

  final result = await Process.run(
    'secret-tool',
    ['lookup', 'service', service, 'account', key],
    runInShell: false,
  );

  if (result.exitCode != 0) {
    throw KeysyncException(
      'Failed to retrieve secret from Linux keyring: ${result.stderr}',
    );
  }

  return (result.stdout as String).trim();
}

/// Retrieve secret from Windows Credential Manager using PowerShell
Future<String> _getWindows(
  String key, {
  String? project,
  String? environment,
}) async {
  // Build tagged credential target name (v2 format)
  final scope = _buildScope(project: project, environment: environment);
  final targetName = _buildWindowsTargetName(
    scope: scope,
    project: project,
    environment: environment,
    key: key,
  );

  // PowerShell command to retrieve credential
  final psCommand = '''
\$cred = Get-StoredCredential -Target "$targetName" -Type Generic -ErrorAction SilentlyContinue
if (\$null -eq \$cred) { exit 1 }
\$ptr = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR(\$cred.Password)
try {
  [System.Runtime.InteropServices.Marshal]::PtrToStringBSTR(\$ptr)
} finally {
  [System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR(\$ptr)
}
''';

  final result = await Process.run(
    'powershell.exe',
    ['-NoProfile', '-Command', psCommand],
    runInShell: false,
  );

  if (result.exitCode != 0) {
    // Try legacy underscore-delimited format for backward compatibility
    final legacyTarget = _buildLegacyWindowsTargetName(
      project: project,
      environment: environment,
      key: key,
    );

    final legacyPsCommand = '''
\$cred = Get-StoredCredential -Target "$legacyTarget" -Type Generic -ErrorAction SilentlyContinue
if (\$null -eq \$cred) { exit 1 }
\$ptr = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR(\$cred.Password)
try {
  [System.Runtime.InteropServices.Marshal]::PtrToStringBSTR(\$ptr)
} finally {
  [System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR(\$ptr)
}
''';

    final legacyResult = await Process.run(
      'powershell.exe',
      ['-NoProfile', '-Command', legacyPsCommand],
      runInShell: false,
    );

    if (legacyResult.exitCode != 0) {
      throw KeysyncException(
        'Failed to retrieve secret from Windows Credential Manager',
      );
    }

    return (legacyResult.stdout as String).trim();
  }

  return (result.stdout as String).trim();
}

/// Build the service name for macOS/Linux keychains
String _buildServiceName({String? project, String? environment}) {
  if (project != null && environment != null) {
    return 'keysync/project/$project/env/$environment';
  } else if (project != null) {
    return 'keysync/project/$project';
  } else {
    return 'keysync/global';
  }
}

/// Build scope identifier for Windows
String _buildScope({String? project, String? environment}) {
  if (project != null && environment != null) {
    return 'env';
  } else if (project != null) {
    return 'project';
  } else {
    return 'global';
  }
}

/// Build Windows credential target name (v2 tagged format)
String _buildWindowsTargetName({
  required String scope,
  String? project,
  String? environment,
  required String key,
}) {
  final parts = ['keysync'];
  parts.add('s=$scope');

  if (project != null) {
    parts.add('p=${_percentEncode(project)}');
  }

  if (environment != null) {
    parts.add('e=${_percentEncode(environment)}');
  }

  parts.add('k=${_percentEncode(key)}');

  return parts.join('|');
}

/// Build legacy Windows credential target name (underscore-delimited)
String _buildLegacyWindowsTargetName({
  String? project,
  String? environment,
  required String key,
}) {
  if (project != null && environment != null) {
    return 'keysync_project_${project}_env_${environment}_$key';
  } else if (project != null) {
    return 'keysync_project_${project}_$key';
  } else {
    return 'keysync_global_$key';
  }
}

/// Percent-encode a string for Windows credential target names
String _percentEncode(String input) {
  final buffer = StringBuffer();
  for (final char in input.runes) {
    final str = String.fromCharCode(char);
    // Encode anything that's not alphanumeric or hyphen
    if (RegExp(r'^[a-zA-Z0-9\-]$').hasMatch(str)) {
      buffer.write(str);
    } else {
      // Percent-encode the UTF-8 bytes
      for (final byte in str.codeUnits) {
        buffer
            .write('%${byte.toRadixString(16).toUpperCase().padLeft(2, '0')}');
      }
    }
  }
  return buffer.toString();
}
