.PHONY: default clean

default:
	cd httpx-static && $(MAKE)

clean:
	cd httpx-static && $(MAKE) clean
