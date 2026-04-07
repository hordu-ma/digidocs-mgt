from .core.config import settings
from .services.dispatcher import WorkerDispatcher


def main() -> None:
    dispatcher = WorkerDispatcher()
    if settings.worker_mode == "once":
        dispatcher.describe_startup()
        return

    dispatcher.run_forever()


if __name__ == "__main__":
    main()
