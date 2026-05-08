package com.gosx.nativekit

import android.content.Context
import android.content.SharedPreferences
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

class GSXSecureStorageException(
    message: String,
    cause: Throwable? = null,
) : RuntimeException(message, cause)

class GSXKeystoreTokenStore(
    context: Context,
    private val keyAlias: String = DEFAULT_KEY_ALIAS,
    private val preferenceName: String = DEFAULT_PREFERENCE_NAME,
    private val tokenKey: String = DEFAULT_TOKEN_KEY,
    private val refreshHandler: (suspend () -> String?)? = null,
) : GSXMutableTokenStore, GSXRefreshableTokenStore {
    private val appContext: Context = context.applicationContext
    private val lock = Any()

    private val preferences: SharedPreferences
        get() = appContext.getSharedPreferences(preferenceName, Context.MODE_PRIVATE)

    override suspend fun token(): String? = synchronized(lock) {
        readToken()
    }

    override suspend fun setToken(token: String?) {
        synchronized(lock) {
            writeToken(token)
        }
    }

    override suspend fun clearToken() {
        setToken(null)
    }

    override suspend fun refreshToken(): String? {
        val refreshed = refreshHandler?.invoke() ?: return null
        setToken(refreshed)
        return refreshed
    }

    private fun readToken(): String? {
        val iv = preferences.getString("${tokenKey}.iv", null)?.let(::decodeBase64) ?: return null
        val ciphertext = preferences.getString("${tokenKey}.ciphertext", null)?.let(::decodeBase64) ?: return null
        return try {
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.DECRYPT_MODE, secretKey(), GCMParameterSpec(GCM_TAG_BITS, iv))
            String(cipher.doFinal(ciphertext), Charsets.UTF_8)
        } catch (error: Throwable) {
            throw GSXSecureStorageException("Unable to decrypt GSX token", error)
        }
    }

    private fun writeToken(token: String?) {
        if (token == null) {
            preferences.edit()
                .remove("${tokenKey}.iv")
                .remove("${tokenKey}.ciphertext")
                .apply()
            return
        }

        try {
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.ENCRYPT_MODE, secretKey())
            val ciphertext = cipher.doFinal(token.toByteArray(Charsets.UTF_8))
            preferences.edit()
                .putString("${tokenKey}.iv", encodeBase64(cipher.iv))
                .putString("${tokenKey}.ciphertext", encodeBase64(ciphertext))
                .apply()
        } catch (error: Throwable) {
            throw GSXSecureStorageException("Unable to encrypt GSX token", error)
        }
    }

    private fun secretKey(): SecretKey {
        val keyStore = KeyStore.getInstance(KEYSTORE_PROVIDER).apply {
            load(null)
        }
        val existing = keyStore.getKey(keyAlias, null) as? SecretKey
        if (existing != null) {
            return existing
        }

        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, KEYSTORE_PROVIDER)
        val spec = KeyGenParameterSpec.Builder(
            keyAlias,
            KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
        )
            .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
            .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
            .setKeySize(256)
            .setRandomizedEncryptionRequired(true)
            .build()
        generator.init(spec)
        return generator.generateKey()
    }

    private fun encodeBase64(bytes: ByteArray): String =
        Base64.encodeToString(bytes, Base64.NO_WRAP)

    private fun decodeBase64(value: String): ByteArray =
        Base64.decode(value, Base64.NO_WRAP)

    companion object {
        public const val DEFAULT_KEY_ALIAS = "gosx.native.token"
        public const val DEFAULT_PREFERENCE_NAME = "gosx_native_secure_storage"
        public const val DEFAULT_TOKEN_KEY = "token"

        private const val KEYSTORE_PROVIDER = "AndroidKeyStore"
        private const val TRANSFORMATION = "AES/GCM/NoPadding"
        private const val GCM_TAG_BITS = 128
    }
}
