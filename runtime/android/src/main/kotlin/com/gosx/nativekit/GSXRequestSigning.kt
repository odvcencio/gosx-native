package com.gosx.nativekit

import java.util.UUID
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

class GSXRequestSigningException(
    message: String,
) : RuntimeException(message)

interface GSXSigningKeyStore {
    suspend fun signingKey(): ByteArray?
}

class GSXMemorySigningKeyStore(
    initialKey: ByteArray? = null,
) : GSXSigningKeyStore {
    @Volatile
    private var currentKey: ByteArray? = initialKey?.copyOf()

    constructor(initialKey: String) : this(initialKey.toByteArray(Charsets.UTF_8))

    override suspend fun signingKey(): ByteArray? = currentKey?.copyOf()

    fun setSigningKey(key: ByteArray?) {
        currentKey = key?.copyOf()
    }

    fun setSigningKey(key: String?) {
        currentKey = key?.toByteArray(Charsets.UTF_8)
    }
}

class GSXRequestSigningOptions(
    val keyID: String? = null,
    val requireKey: Boolean = true,
    val timestampHeader: String = "X-GSX-Timestamp",
    val nonceHeader: String = "X-GSX-Nonce",
    val bodyHashHeader: String = "X-GSX-Body-SHA256",
    val signatureHeader: String = "X-GSX-Signature",
    val keyIDHeader: String = "X-GSX-Key-ID",
    val timestampProvider: () -> String = { System.currentTimeMillis().toString() },
    val nonceProvider: () -> String = { UUID.randomUUID().toString() },
)

class GSXRequestSigningTransport(
    private val base: GSXTransport,
    private val keyStore: GSXSigningKeyStore,
    private val options: GSXRequestSigningOptions = GSXRequestSigningOptions(),
) : GSXTransport {
    override suspend fun send(request: GSXRequest): GSXResponse {
        if (request.headers.keys.any { it.equals(options.signatureHeader, ignoreCase = true) }) {
            return base.send(request)
        }

        val key = keyStore.signingKey()
        if (key == null || key.isEmpty()) {
            if (options.requireKey) {
                throw GSXRequestSigningException("Missing GSX request signing key")
            }
            return base.send(request)
        }

        val timestamp = options.timestampProvider()
        val nonce = options.nonceProvider()
        val bodyHash = sha256Hex(request.body ?: ByteArray(0))
        val canonical = canonicalPayload(
            method = request.method,
            path = request.path,
            timestamp = timestamp,
            nonce = nonce,
            bodyHash = bodyHash,
        )
        val signedHeaders = request.headers.toMutableMap()
        signedHeaders[options.timestampHeader] = timestamp
        signedHeaders[options.nonceHeader] = nonce
        signedHeaders[options.bodyHashHeader] = bodyHash
        signedHeaders[options.signatureHeader] = hmacHex(canonical, key)
        val keyID = options.keyID?.trim()
        if (!keyID.isNullOrEmpty()) {
            signedHeaders[options.keyIDHeader] = keyID
        }
        return base.send(request.copy(headers = signedHeaders))
    }

    private fun canonicalPayload(
        method: String,
        path: String,
        timestamp: String,
        nonce: String,
        bodyHash: String,
    ): String = listOf(
        method.uppercase(),
        path,
        timestamp,
        nonce,
        bodyHash,
    ).joinToString("\n")

    private fun sha256Hex(bytes: ByteArray): String =
        java.security.MessageDigest.getInstance("SHA-256")
            .digest(bytes)
            .toHex()

    private fun hmacHex(canonical: String, key: ByteArray): String {
        val mac = Mac.getInstance("HmacSHA256")
        mac.init(SecretKeySpec(key, "HmacSHA256"))
        return mac.doFinal(canonical.toByteArray(Charsets.UTF_8)).toHex()
    }

    private fun ByteArray.toHex(): String {
        val out = CharArray(size * 2)
        for (index in indices) {
            val value = this[index].toInt() and 0xff
            out[index * 2] = HEX[value ushr 4]
            out[index * 2 + 1] = HEX[value and 0x0f]
        }
        return String(out)
    }

    companion object {
        private val HEX = "0123456789abcdef".toCharArray()
    }
}
