START = 1
STOP = 2


def run(command):
    if command == START:
        return 1
    if command == STOP:
        return 0
    return -1
