import Foundation
import Security

public enum GSXSecureStorageError: Error, Equatable {
    case invalidTokenData
    case keychainStatus(OSStatus)
}

public actor GSXKeychainTokenStore: GSXMutableTokenStore, GSXRefreshableTokenStore {
    private let service: String
    private let account: String
    private let accessGroup: String?
    private let refreshHandler: GSXTokenRefreshHandler?

    public init(
        service: String = "gosx.native.token",
        account: String = "default",
        accessGroup: String? = nil,
        refresh: GSXTokenRefreshHandler? = nil
    ) {
        self.service = service
        self.account = account
        self.accessGroup = accessGroup
        self.refreshHandler = refresh
    }

    public func token() async throws -> String? {
        var query = baseQuery()
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne

        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        if status == errSecItemNotFound {
            return nil
        }
        guard status == errSecSuccess else {
            throw GSXSecureStorageError.keychainStatus(status)
        }
        guard let data = item as? Data, let token = String(data: data, encoding: .utf8) else {
            throw GSXSecureStorageError.invalidTokenData
        }
        return token
    }

    public func setToken(_ token: String?) async throws {
        guard let token else {
            try deleteToken()
            return
        }
        guard let data = token.data(using: .utf8) else {
            throw GSXSecureStorageError.invalidTokenData
        }

        let query = baseQuery()
        let attributes: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
        ]
        var addQuery = query
        for (name, value) in attributes {
            addQuery[name] = value
        }

        let status = SecItemAdd(addQuery as CFDictionary, nil)
        if status == errSecDuplicateItem {
            let updateStatus = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
            guard updateStatus == errSecSuccess else {
                throw GSXSecureStorageError.keychainStatus(updateStatus)
            }
            return
        }
        guard status == errSecSuccess else {
            throw GSXSecureStorageError.keychainStatus(status)
        }
    }

    public func clearToken() async throws {
        try deleteToken()
    }

    public func refreshToken() async throws -> String? {
        guard let refreshHandler else {
            return nil
        }
        let token = try await refreshHandler()
        try await setToken(token)
        return token
    }

    private func deleteToken() throws {
        let status = SecItemDelete(baseQuery() as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw GSXSecureStorageError.keychainStatus(status)
        }
    }

    private func baseQuery() -> [String: Any] {
        var query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        if let accessGroup, !accessGroup.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            query[kSecAttrAccessGroup as String] = accessGroup
        }
        return query
    }
}
