import logging

from .core.config import settings
from .services.dispatcher import WorkerDispatcher

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)


def main() -> None:
    dispatcher = WorkerDispatcher()
    if settings.worker_mode == "once":
        dispatcher.describe_startup()
        return

    dispatcher.run_forever()


if __name__ == "__main__":
    main()
