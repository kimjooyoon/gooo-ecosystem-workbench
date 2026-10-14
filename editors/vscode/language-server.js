'use strict';

// Starting and disposing clients must finish before another client is created.
// Settings can change during either operation; only the latest path is needed.
function createLanguageServerController(createClient, reportError) {
  let compiler;
  let requestVersion = 0;
  let lastAttemptVersion = 0;
  let current;
  let closed = false;
  let pending = Promise.resolve();

  async function reconcile() {
    while (true) {
      if (current && !closed && current.compiler === compiler &&
          (current.started || current.requestVersion === requestVersion)) return;
      if (current && (closed || current.compiler !== compiler || !current.started)) {
        const needsStop = current.started || current.client.needsStop();
        try {
          await current.client.dispose();
        } catch (error) {
          // LanguageClient 10 marks a rejected start as StartFailed. Its dispose
          // still schedules process cleanup, but rejects because there is no running
          // connection to stop. A running client's disposal must succeed.
          if (needsStop) throw new Error(`Could not stop the Gooo language server: ${errorText(error)}`);
        }
        current = undefined;
      }
      if (closed || current) return;
      if (lastAttemptVersion === requestVersion) return;
      lastAttemptVersion = requestVersion;

      const next = { compiler, requestVersion, client: createClient(compiler), started: false };
      current = next;
      try {
        await next.client.start();
        next.started = true;
      } catch (error) {
        if (!closed && next.compiler === compiler && next.requestVersion === requestVersion) {
          reportError(`Could not start the Gooo language server: ${errorText(error)}`);
          return;
        }
      }
      if (!closed && next.started && next.compiler === compiler) return;
    }
  }

  function enqueue() {
    pending = pending.then(reconcile).catch((error) => reportError(errorText(error)));
    return pending;
  }

  return {
    updateCompiler(command) {
      if (closed) return pending;
      compiler = command;
      requestVersion += 1;
      return enqueue();
    },
    dispose() {
      if (closed) return pending;
      closed = true;
      return enqueue();
    }
  };
}

function errorText(error) {
  return String(error?.message ?? error);
}

module.exports = { createLanguageServerController };
