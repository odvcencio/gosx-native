package com.gosx.nativekit

enum class GSXCrashSeverity {
    Handled,
    Error,
    Fatal,
}

data class GSXCrashReport(
    val name: String,
    val message: String,
    val severity: GSXCrashSeverity = GSXCrashSeverity.Error,
    val stack: String? = null,
    val attributes: Map<String, String> = emptyMap(),
    val timestampMillis: Long = System.currentTimeMillis(),
) {
    companion object {
        fun fromThrowable(
            error: Throwable,
            severity: GSXCrashSeverity = GSXCrashSeverity.Error,
            attributes: Map<String, String> = emptyMap(),
        ): GSXCrashReport = GSXCrashReport(
            name = error::class.java.name,
            message = error.message ?: error.toString(),
            severity = severity,
            stack = error.stackTraceToString(),
            attributes = attributes,
        )
    }
}

fun interface GSXCrashReporter {
    fun record(report: GSXCrashReport)
}

object GSXNoopCrashReporter : GSXCrashReporter {
    override fun record(report: GSXCrashReport) = Unit
}

class GSXMemoryCrashReporter : GSXCrashReporter {
    private val lock = Any()
    private val reports = mutableListOf<GSXCrashReport>()

    override fun record(report: GSXCrashReport) {
        synchronized(lock) {
            reports += report
        }
    }

    fun recordedReports(): List<GSXCrashReport> = synchronized(lock) {
        reports.toList()
    }

    fun reset() {
        synchronized(lock) {
            reports.clear()
        }
    }
}

class GSXDiagnosticsCrashReporter(
    private val diagnostics: GSXDiagnosticsRecorder = GSXDiagnostics,
) : GSXCrashReporter {
    override fun record(report: GSXCrashReport) {
        diagnostics.record(
            GSXDiagnosticEvent(
                category = "crash",
                name = report.name,
                level = diagnosticLevel(report.severity),
                message = report.message,
                attributes = report.attributes +
                    ("severity" to report.severity.name) +
                    ("has_stack" to (report.stack != null).toString()),
                timestampMillis = report.timestampMillis,
            ),
        )
    }

    private fun diagnosticLevel(severity: GSXCrashSeverity): GSXDiagnosticLevel =
        when (severity) {
            GSXCrashSeverity.Handled -> GSXDiagnosticLevel.Warning
            GSXCrashSeverity.Error, GSXCrashSeverity.Fatal -> GSXDiagnosticLevel.Error
        }
}

object GSXCrashReporting : GSXCrashReporter {
    @Volatile
    private var reporter: GSXCrashReporter = GSXNoopCrashReporter
    private val handlerLock = Any()
    private var installedHandler: Thread.UncaughtExceptionHandler? = null

    fun configure(reporter: GSXCrashReporter) {
        this.reporter = reporter
    }

    override fun record(report: GSXCrashReport) {
        reporter.record(report)
    }

    fun record(
        error: Throwable,
        severity: GSXCrashSeverity = GSXCrashSeverity.Error,
        attributes: Map<String, String> = emptyMap(),
    ) {
        record(GSXCrashReport.fromThrowable(error, severity, attributes))
    }

    fun record(
        name: String,
        message: String,
        severity: GSXCrashSeverity = GSXCrashSeverity.Error,
        stack: String? = null,
        attributes: Map<String, String> = emptyMap(),
    ) {
        record(
            GSXCrashReport(
                name = name,
                message = message,
                severity = severity,
                stack = stack,
                attributes = attributes,
            ),
        )
    }

    inline fun <T> capture(
        severity: GSXCrashSeverity = GSXCrashSeverity.Error,
        attributes: Map<String, String> = emptyMap(),
        block: () -> T,
    ): T {
        return try {
            block()
        } catch (error: Throwable) {
            record(error, severity, attributes)
            throw error
        }
    }

    suspend inline fun <T> captureSuspend(
        severity: GSXCrashSeverity = GSXCrashSeverity.Error,
        attributes: Map<String, String> = emptyMap(),
        crossinline block: suspend () -> T,
    ): T {
        return try {
            block()
        } catch (error: Throwable) {
            record(error, severity, attributes)
            throw error
        }
    }

    fun installDefaultUncaughtExceptionHandler(
        delegate: Thread.UncaughtExceptionHandler? = Thread.getDefaultUncaughtExceptionHandler(),
    ) {
        synchronized(handlerLock) {
            if (Thread.getDefaultUncaughtExceptionHandler() === installedHandler) {
                return
            }
            val handler = Thread.UncaughtExceptionHandler { thread, error ->
                record(
                    error = error,
                    severity = GSXCrashSeverity.Fatal,
                    attributes = mapOf(
                        "source" to "Thread.UncaughtExceptionHandler",
                        "thread" to thread.name,
                    ),
                )
                delegate?.uncaughtException(thread, error)
            }
            installedHandler = handler
            Thread.setDefaultUncaughtExceptionHandler(handler)
        }
    }
}
