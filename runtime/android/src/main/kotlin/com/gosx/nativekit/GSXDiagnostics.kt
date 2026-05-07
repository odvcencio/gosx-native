package com.gosx.nativekit

enum class GSXDiagnosticLevel {
    Debug,
    Info,
    Warning,
    Error,
}

data class GSXDiagnosticEvent(
    val category: String,
    val name: String,
    val level: GSXDiagnosticLevel = GSXDiagnosticLevel.Info,
    val message: String,
    val attributes: Map<String, String> = emptyMap(),
    val timestampMillis: Long = System.currentTimeMillis(),
)

fun interface GSXDiagnosticsRecorder {
    fun record(event: GSXDiagnosticEvent)
}

fun interface GSXDiagnosticsSink {
    fun record(event: GSXDiagnosticEvent)
}

object GSXNoopDiagnosticsSink : GSXDiagnosticsSink {
    override fun record(event: GSXDiagnosticEvent) = Unit
}

class GSXMemoryDiagnosticsSink : GSXDiagnosticsSink {
    private val lock = Any()
    private val recordedEvents = mutableListOf<GSXDiagnosticEvent>()

    override fun record(event: GSXDiagnosticEvent) {
        synchronized(lock) {
            recordedEvents += event
        }
    }

    fun events(): List<GSXDiagnosticEvent> = synchronized(lock) {
        recordedEvents.toList()
    }

    fun reset() {
        synchronized(lock) {
            recordedEvents.clear()
        }
    }
}

object GSXDiagnostics : GSXDiagnosticsRecorder {
    @Volatile
    private var sink: GSXDiagnosticsSink = GSXNoopDiagnosticsSink

    fun configure(sink: GSXDiagnosticsSink) {
        this.sink = sink
    }

    override fun record(event: GSXDiagnosticEvent) {
        sink.record(event)
    }

    fun record(
        category: String,
        name: String,
        level: GSXDiagnosticLevel = GSXDiagnosticLevel.Info,
        message: String,
        attributes: Map<String, String> = emptyMap(),
    ) {
        record(
            GSXDiagnosticEvent(
                category = category,
                name = name,
                level = level,
                message = message,
                attributes = attributes,
            ),
        )
    }
}
