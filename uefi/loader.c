#include <efi.h>
#include <efilib.h>

#define ELF_HEADER_SIZE 64
#define ELF_MAGIC 0x464C457F
#define ET_EXEC 0x02
#define PT_LOAD 1

static EFI_STATUS loadKernel(EFI_HANDLE imgHandle, EFI_SYSTEM_TABLE *sysTab, char **kernelRet, UINTN *kernelSizeRet) {
	EFI_STATUS res;
	EFI_LOADED_IMAGE_PROTOCOL *image;
	EFI_SIMPLE_FILE_SYSTEM_PROTOCOL *fs;
	EFI_FILE_PROTOCOL *root, *kernelFile;
	UINTN bufSize;
	EFI_FILE_INFO *fileInfo;
	UINTN kernelSize;
	VOID *kernel;

	res = uefi_call_wrapper(sysTab->BootServices->HandleProtocol, 3,
		imgHandle,
		&gEfiLoadedImageProtocolGuid,
		&image);
	if (res != EFI_SUCCESS) {
		Print(L"HandleProtocol(EFI_LOADED_IMAGE_PROTOCOL) failed: %d\n", res);
		return res;
	}
	Print(L"Found image protocol\n");
	res = uefi_call_wrapper(sysTab->BootServices->HandleProtocol, 3,
		image->DeviceHandle,
		&gEfiSimpleFileSystemProtocolGuid,
		&fs);
	if (res != EFI_SUCCESS) {
		Print(L"HandleProtocol(EFI_SIMPLE_FILE_SYSTEM_PROTOCOL) failed: %d\n", res);
		return res;
	}
	Print(L"Found file system\n");
	res = uefi_call_wrapper(fs->OpenVolume, 2, fs, &root);
	if (res != EFI_SUCCESS) {
		Print(L"OpenVolume failed: %d\n", res);
		return res;
	}
	Print(L"Opened file system root\n");
	res = uefi_call_wrapper(root->Open, 5, root, &kernelFile, L"\\KERNEL.ELF", EFI_FILE_MODE_READ, 0);
	uefi_call_wrapper(root->Close, 1, root);
	if (res != EFI_SUCCESS) {
		Print(L"Open(\"\\KERNEL.ELF\") failed: %d\n", res);
		return res;
	}
	bufSize = 0;
	res = uefi_call_wrapper(kernelFile->GetInfo, 4, kernelFile, &gEfiFileInfoGuid, &bufSize, NULL);
	if (res != EFI_BUFFER_TOO_SMALL) {
		Print(L"kernel->GetInfo failed: %d\n", res);
		uefi_call_wrapper(kernelFile->Close, 1, kernelFile);
		return res;
	}
	fileInfo = AllocatePool(bufSize);
	if (fileInfo == NULL) {
		Print(L"AllocPool(%d) failed\n", bufSize);
		return EFI_OUT_OF_RESOURCES;
	}
	res = uefi_call_wrapper(kernelFile->GetInfo, 4, kernelFile, &gEfiFileInfoGuid, &bufSize, fileInfo);
	kernelSize = fileInfo->FileSize;
	FreePool(fileInfo);
	if (res != EFI_SUCCESS) {
		Print(L"kernel->GetInfo failed: %d\n", res);
		uefi_call_wrapper(kernelFile->Close, 1, kernelFile);
		return res;
	}
	Print(L"Found kernel, size %d\n", kernelSize);
	if (kernelSize < ELF_HEADER_SIZE) {
		Print(L"kernel image too small (%d)\n", kernelSize);
		uefi_call_wrapper(kernelFile->Close, 1, kernelFile);
		return res;
	}
	kernel = AllocatePool(kernelSize);
	if (kernel == NULL) {
		uefi_call_wrapper(kernelFile->Close, 1, kernelFile);
		Print(L"AllocPool(%d) failed\n", kernelSize);
		return EFI_OUT_OF_RESOURCES;
	}
	res = uefi_call_wrapper(kernelFile->Read, 3, kernelFile, &kernelSize, kernel);
	uefi_call_wrapper(kernelFile->Close, 1, kernelFile);
	if (res != EFI_SUCCESS) {
		FreePool(kernel);
		Print(L"kernel->Read failed: %d\n", res);
		return res;
	}
	*kernelRet = kernel;
	*kernelSizeRet = kernelSize;
	return EFI_SUCCESS;
}

EFI_STATUS
EFIAPI
efi_main (EFI_HANDLE imgHandle, EFI_SYSTEM_TABLE *sysTab) {
	EFI_STATUS res;
	char *kernel;
	UINTN kernelSize;

	InitializeLib(imgHandle, sysTab);
	uefi_call_wrapper(sysTab->ConOut->ClearScreen, 1, sysTab->ConOut);
	Print(L"Booting...\n");
	res = loadKernel(imgHandle, sysTab, &kernel, &kernelSize);
	if (res != EFI_SUCCESS) {
		FreePool(kernel);
		return EFI_LOAD_ERROR;
	}

	int32_t magic = *(int32_t *)(kernel + 0);
	if (magic != ELF_MAGIC) {
		FreePool(kernel);
		Print(L"kernel ELF magic 0x%x, expected 0x%x\n", magic, ELF_MAGIC);
		return res;
	}
	if (kernel[4] != 2 /* 64-bit*/ || kernel[16] != ET_EXEC) {
		FreePool(kernel);
		Print(L"kernel is not a 64-bit executable\n");
		return res;
	}
	uint64_t entryAddr = *(uint64_t *)(kernel + 24);
	Print(L"kernel entry point: 0x%lx\n", entryAddr);
	uint16_t phdrSize = *(uint16_t *)(kernel + 54);
	if (phdrSize < 56) {
		FreePool(kernel);
		Print(L"kernel program header entry size too small (%d)\n", phdrSize);
		return res;
	}
	uint64_t phdrOff = *(uint64_t *)(kernel + 32);
	uint16_t phdrCount = *(uint16_t *)(kernel + 56);
	Print(L"PHDR: offset %ld entry size %d count %d\n", phdrOff, phdrSize, phdrCount);
	char *phdrTab = kernel + phdrOff;
	for (int i = 0; i < phdrCount; i++) {
		char *phdr = phdrTab + i*phdrSize;
		uint32_t type = *(uint32_t *)(phdr + 0);
		if (type != PT_LOAD) {
			continue;
		}
		uint64_t p_off = *(uint64_t *)(phdr + 8);
		uint64_t p_vaddr = *(uint64_t *)(phdr + 16);
		uint64_t p_filesz = *(uint64_t *)(phdr + 32);
		uint64_t p_memsz = *(uint64_t *)(phdr + 40);
		void *dest = (void *)p_vaddr;
		void *src = (void *)(kernel + p_off);
		Print(L"PT_LOAD: off 0x%lx vaddr 0x%lx filesz 0x%lx memsz 0x%lx\n", p_off, p_vaddr, p_filesz, p_memsz);
		uefi_call_wrapper(sysTab->BootServices->SetMem, 3, dest, p_memsz, 0);
		uefi_call_wrapper(sysTab->BootServices->CopyMem, 3, dest, src, p_filesz);
	}

	// Go side reads the kernel image.
	//FreePool(kernel);

	Print(L"Exiting boot services and jumping to entry 0x%lx\n", entryAddr);
	UINTN mapKey;
	UINTN mmapSize = 0;
	EFI_MEMORY_DESCRIPTOR *mmap = NULL;
	UINTN descSize = 0;
	UINT32 descVer;
	res = uefi_call_wrapper(sysTab->BootServices->GetMemoryMap, 5, &mmapSize, mmap, &mapKey, &descSize, &descVer);
	if (res != EFI_BUFFER_TOO_SMALL && res != EFI_SUCCESS) {
		Print(L"GetMemoryMap failed: %d\n", res);
		for (;;) {}
		return EFI_LOAD_ERROR;
	}
	// Allocate space for memory map, plus a few entries for the map
	// to grow because of the allocation.
	mmapSize += 10*descSize;
	mmap = AllocatePool(mmapSize);
	if (mmap == NULL) {
		Print(L"AllocPool(%d) failed\n", mmapSize);
		return EFI_OUT_OF_RESOURCES;
	}
	res = uefi_call_wrapper(sysTab->BootServices->GetMemoryMap, 5, &mmapSize, mmap, &mapKey, &descSize, &descVer);
	if (res != EFI_SUCCESS) {
		Print(L"GetMemoryMap failed: %d\n", res);
		return EFI_LOAD_ERROR;
	}

	res = uefi_call_wrapper(sysTab->BootServices->ExitBootServices, 2, imgHandle, mapKey);
	if (res != EFI_SUCCESS) {
		Print(L"ExitBootServices failed: %d\n", res);
		return EFI_LOAD_ERROR;
	}

	typedef void (*entryFunc)(uint64_t mmapSize, uint64_t descSize, uint64_t kernelImageSize, EFI_MEMORY_DESCRIPTOR *mmap, void *kernelImage);

	entryFunc entry = (entryFunc)entryAddr;
	entry(mmapSize, descSize, kernelSize, mmap, kernel);
	return EFI_LOAD_ERROR;
}
