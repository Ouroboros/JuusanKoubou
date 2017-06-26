#pragma comment(linker,"/SECTION:.text,ERW /MERGE:.rdata=.text /MERGE:.data=.text")
#pragma comment(linker,"/SECTION:.Amano,ERW /MERGE:.text=.Amano")

#include "yu-ris.h"
#include "zlib/zlib.h"
#include "gal_common.h"
#include "my_commsrc.h"

Void EncryptFileName(PByte pbBuffer, UInt32 uSize, Byte Key)
{
    while (uSize--)
        pbBuffer[uSize] = (-pbBuffer[uSize] - 1) ^ Key;
}

UInt32 HashFileName(LPVoid lpFileName, Int32 Length)
{
    PByte  pbFileName = (PByte)lpFileName;
    UInt32 Hash = -1;

    while (Length--)
    {
        Hash = (Hash >> 8) ^ HashTable[(*pbFileName++ ^ Hash) & 0xFF];
    }

    return ~Hash;
}

Void EncryptScript(LPVoid lpBuffer, UInt32 Size, UInt32 Version)
{
    PByte pbFile, pbBuffer;
    ULong Mask;

    switch (Version)
    {
        case 0x107:
            Mask = TAG4(0x07B4024A);
            break;

        case 0x122:
            Mask = 0x96AC6FD3;
            break;
    }

    pbFile = (PByte)lpBuffer;
    pbBuffer = pbFile;
    if (*(PUInt32)pbFile == TAG4('YSTB')) do
    {
        ULong ulPart1, ulPart2;

        if (Version == 0x107)
        {
            ulPart1 = *(PULong)(pbFile + 0x8);
            ulPart2 = *(PULong)(pbFile + 0xC);
        }
        else if (Version == 0x122)
        {
            ulPart1 = *(PULong)(pbFile + 0xC);
            ulPart2 = *(PULong)(pbFile + 0x10);
        }

        pbBuffer = pbFile + 0x20;

        while (ulPart1 > 3)
        {
            *(PULong)pbBuffer ^= Mask;
            pbBuffer += 4;
            ulPart1 -= 4;
        }
        for (ULong i = 0, j = ulPart1; i != j; ++i)
            *pbBuffer++ ^= Mask >> (i * 8);

        while (ulPart2 > 3)
        {
            *(PULong)pbBuffer ^= Mask;
            pbBuffer += 4;
            ulPart2 -= 4;
        }
        for (ULong i = 0, j = ulPart2; i != j; ++i)
            *pbBuffer++ ^= Mask >> (i * 8);

        if (Version == 0x107)
            break;

        ulPart1 = *(PULong)(pbFile + 0x14);
        ulPart2 = *(PULong)(pbFile + 0x18);

        while (ulPart1 > 3)
        {
            *(PULong)pbBuffer ^= Mask;
            pbBuffer += 4;
            ulPart1 -= 4;
        }
        for (ULong i = 0, j = ulPart1; i != j; ++i)
            *pbBuffer++ ^= Mask >> (i * 8);

        while (ulPart2 > 3)
        {
            *(PULong)pbBuffer ^= Mask;
            pbBuffer += 4;
            ulPart2 -= 4;
        }
        for (ULong i = 0, j = ulPart2; i != j; ++i)
            *pbBuffer++ ^= Mask >> (i * 8);

    } while (0);
}

UInt GetPacFileType(PWChar pszFileName)
{
    WChar *szExtension[] = 
    {
        NULL, L".bmp", L".png", L".jpg", L".gif", L".wav", L".ogg", L".psd", 
    };

    for (UInt i = 0; i != countof(szExtension); ++i)
        if (!lstrcmpiW(szExtension[i], findextw(pszFileName)))
            return i;
    return 0;
}

Void ReleaseGlobalData()
{
}

int CDECL compare(const void *, const void *);

Void CALLBACK PackFiles(SPackFileInfo *pPackFileInfo, UInt32 uCount, PCWChar pszOutput, PCWChar pszFullInputPath, PCWChar pszParam, FShowStatus ShowStatus)
{
    CMem    m;
    HANDLE  hFile, hFilePack;
    UInt32  uBufferSize, uCompressSize;
    DWORD   dwRead;
    WChar   szPath[MAX_PATH];
    LPVoid  lpBuffer, lpCompressBuffer;
    Large_Integer liOffset;
    SPackFileInfo *pInfo;
    SMyPacFileInfo  *pIndex, *pPackIndex;
    SPacHeader header;

    hFilePack = CreateFileW(pszOutput, 
                    GENERIC_READ|GENERIC_WRITE, 
                    FILE_SHARE_READ|FILE_SHARE_WRITE, 
                    NULL, 
                    CREATE_ALWAYS, 
                    FILE_ATTRIBUTE_NORMAL, 
                    NULL);
    if (hFilePack == INVALID_HANDLE_VALUE)
        return;

    GetCurrentDirectoryW(countof(szPath), szPath);
    SetCurrentDirectoryW(pszFullInputPath);

    pIndex = (SMyPacFileInfo *)m.Alloc(uCount * sizeof(*pIndex));
    if (pIndex == NULL)
        return;

    header.ver = 0x122;
    if (pszParam)
    {
        if (!lstrcmpW(pszParam, L"107"))
            header.ver = 0x107;
    }

    // calc index size and reserve space for it
    uBufferSize = 0x20;
    pInfo = pPackFileInfo;
    for (UInt32 i = 0; i != uCount; ++i, ++pInfo)
    {
        WChar szFile[MAX_PATH];

        uBufferSize += 4 + 1 + sizeof(SPacFileInfo);
        lstrcpyW(szFile, pInfo->pFileName);
        FilterString(szFile, -1);
        uBufferSize += WideCharToMultiByte(CP_GB2312, 0, szFile, -1, 0, 0, 0 ,0) - 1;
    }

    header.entrysize = uBufferSize;
    header.tag = TAG3('YPF');

    uBufferSize = max(uBufferSize, 0x10000);
    uCompressSize = uBufferSize;
    lpBuffer = m.Alloc(uBufferSize);
    lpCompressBuffer = m.Alloc(uCompressSize);

    ZeroMemory((PByte)lpBuffer + 0x10, 0x10);
    WriteFile(hFilePack, lpBuffer, header.entrysize, &dwRead, NULL);

    header.filenum = 0;
    liOffset.QuadPart = header.entrysize;
    header.entrysize -= 0x20;
    pInfo = pPackFileInfo;
    pPackIndex = pIndex;
    for (UInt32 i = 0; i != uCount; ++i, ++pIndex, ++pInfo)
    {
        hFile = CreateFileW(pInfo->pFileName, 
                    GENERIC_READ, 
                    FILE_SHARE_READ, 
                    NULL, 
                    OPEN_EXISTING, 
                    FILE_ATTRIBUTE_NORMAL, 
                    NULL);
        if (hFile == INVALID_HANDLE_VALUE)
        {
            --pIndex;
            continue;
        }

        ZeroMemory(pIndex, sizeof(*pIndex));
        lstrcpyW(pIndex->szFileName, pInfo->pFileName);
        dwRead = GetFileSize(hFile, NULL);
        if (dwRead > uBufferSize)
        {
            uBufferSize = dwRead;
            lpBuffer = m.ReAlloc(lpBuffer, uBufferSize);
        }

        pIndex->uOffset = liOffset.LowPart;
        pIndex->uDecompSize = dwRead;
        if (dwRead)
        {
            ReadFile(hFile, lpBuffer, dwRead, &dwRead, NULL);

            UInt32 v = *(PUInt32)lpBuffer;
            if (v == TAG4('YSTB'))
            {
                EncryptScript(lpBuffer, dwRead, header.ver);
                pIndex->bCompress = True;
            }
            else if (v != TAG4('OggS') &&
                    (v >> 8) != TAG3('PNG') &&
                    (v & 0x00FFFFFF) != 0xFFD8FF)
            {
                pIndex->bCompress = True;
            }

            if (pIndex->bCompress)
            {
                UInt32 Size;

                if (dwRead > uCompressSize)
                {
                    uCompressSize = dwRead;
                    lpCompressBuffer = m.ReAlloc(lpCompressBuffer, uCompressSize);
                }
                memcpy(lpCompressBuffer, lpBuffer, dwRead);
                Size = dwRead;
                dwRead = uBufferSize;
                compress2((PByte)lpBuffer, &dwRead, (PByte)lpCompressBuffer, Size, Z_BEST_COMPRESSION);
            }

            pIndex->uHash2 = adler32(1, (PByte)lpBuffer, dwRead);
            WriteFile(hFilePack, lpBuffer, dwRead, &dwRead, NULL);
            liOffset.QuadPart += dwRead;
        }
        CloseHandle(hFile);

        pIndex->uCompSize = dwRead;

        ++header.filenum;

        if (ShowStatus)
        {
            WChar buf[400];
            swprintf(buf, L"%u / %u : %s", i + 1, uCount, pIndex->szFileName);
            if (!ShowStatus(buf, (i + 1) * 100 / uCount))
                break;
        }
    }

    SetFilePointer(hFilePack, 0, NULL, FILE_BEGIN);
    WriteFile(hFilePack, &header, sizeof(header), &dwRead, NULL);
    SetFilePointer(hFilePack, 0x20, NULL, FILE_BEGIN);

    Byte Key;
    PByte LengthTable;

    switch (header.ver)
    {
        case 0x107:
            Key = 0x40;
            LengthTable = (PByte)LengthTable_107;
            break;

        case 0x122:
        default:
            Key = 0x3F;
            Key = 0xCB; // neko koi
            LengthTable = (PByte)LengthTable_122;
            break;
    }

    qsort(pPackIndex, header.filenum, sizeof(*pIndex), compare);
    pIndex = pPackIndex;
    for (UInt32 i = 0; i != header.filenum; ++i, ++pIndex)
    {
        Char szFile[MAX_PATH];
        UInt32 Length, Len2;
        SPacFileInfo info;
        WChar szFileW[MAX_PATH];

        lstrcpyW(szFileW, pIndex->szFileName);
        FilterString(szFileW, -1);

//        info.cFileType = pIndex->cFileType;
        info.cFileType = GetPacFileType(pIndex->szFileName);
        info.bComp = pIndex->bCompress;
        info.compsize = pIndex->uCompSize;
        info.decompsize = pIndex->uDecompSize;
        info.offset = pIndex->uOffset;
        Length = WideCharToMultiByte(CP_SHIFTJIS, 0, szFileW, -1, szFile, sizeof(szFile), 0, 0);
        --Length;
        for (Len2 = 0; Len2 != 256; ++Len2)
            BREAK_IF(Length == LengthTable[Len2])

        info.hash = HashFileName(szFile, Length);
        WriteFile(hFilePack, &info.hash, 4, &dwRead, NULL);
        WriteFile(hFilePack, &Len2, 1, &dwRead, NULL);
        EncryptFileName((PByte)szFile, Length, Key);
        WriteFile(hFilePack, szFile, Length, &dwRead, NULL);
        info.hash = pIndex->uHash2;
        WriteFile(hFilePack, &info, sizeof(info), &dwRead, NULL);
    }

    CloseHandle(hFilePack);

    m.Free(lpBuffer);
    m.Free(lpCompressBuffer);

    SetCurrentDirectoryW(szPath);

    ReleaseGlobalData();
}

int CDECL compare(const void *v1, const void *v2)
{
    UInt32 len1, len2;
    Char n1[MAX_PATH], n2[MAX_PATH];
    SMyPacFileInfo *p1, *p2;

    p1 = (SMyPacFileInfo *)v1;
    p2 = (SMyPacFileInfo *)v2;
    len1 = WideCharToMultiByte(CP_GB2312, 0, p1->szFileName, -1, n1, sizeof(n1), 0, 0) - 1;
    len2 = WideCharToMultiByte(CP_GB2312, 0, p2->szFileName, -1, n2, sizeof(n2), 0, 0) - 1;

    len1 = HashFileName(n1, len1);
    len2 = HashFileName(n2, len2);

    if (len1 < len2)
        return -1;
    if (len1 == len2)
        return 0;
    return 1;
}

Byte LengthTable_107[256] =
{
    0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8, 0xF7, 0xF6,
    0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0, 0xEF, 0xEE, 0xED, 0xEC,
    0xEB, 0xEA, 0xE9, 0xE8, 0xE7, 0xE6, 0xE5, 0xE4, 0xE3, 0xE2,
    0xE1, 0xE0, 0xDF, 0xDE, 0xDD, 0xDC, 0xDB, 0xDA, 0xD9, 0xD8,
    0xD7, 0xD6, 0xD5, 0xD4, 0xD3, 0xD2, 0xD1, 0xD0, 0xCF, 0xCE,
    0xCD, 0xCC, 0xCB, 0xCA, 0xC9, 0xC8, 0xC7, 0xC6, 0xC5, 0xC4,
    0xC3, 0xC2, 0xC1, 0xC0, 0xBF, 0xBE, 0xBD, 0xBC, 0xBB, 0xBA,
    0xB9, 0xB8, 0xB7, 0xB6, 0xB5, 0xB4, 0xB3, 0xB2, 0xB1, 0xB0,
    0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8, 0xA7, 0xA6,
    0xA5, 0xA4, 0xA3, 0xA2, 0xA1, 0xA0, 0x9F, 0x9E, 0x9D, 0x9C,
    0x9B, 0x9A, 0x99, 0x98, 0x97, 0x96, 0x95, 0x94, 0x93, 0x92,
    0x91, 0x90, 0x8F, 0x8E, 0x8D, 0x8C, 0x8B, 0x8A, 0x89, 0x88,
    0x87, 0x86, 0x85, 0x84, 0x83, 0x82, 0x81, 0x80, 0x7F, 0x7E,
    0x7D, 0x7C, 0x7B, 0x7A, 0x79, 0x78, 0x77, 0x76, 0x75, 0x74,
    0x73, 0x72, 0x71, 0x70, 0x6F, 0x6E, 0x6D, 0x6C, 0x6B, 0x6A,
    0x69, 0x68, 0x67, 0x66, 0x65, 0x64, 0x63, 0x62, 0x61, 0x60,
    0x5F, 0x5E, 0x5D, 0x5C, 0x5B, 0x5A, 0x59, 0x58, 0x57, 0x56,
    0x55, 0x54, 0x53, 0x52, 0x51, 0x50, 0x4F, 0x4E, 0x4D, 0x4C,
    0x4B, 0x4A, 0x49, 0x48, 0x47, 0x46, 0x45, 0x44, 0x43, 0x42,
    0x41, 0x40, 0x3F, 0x3E, 0x3D, 0x3C, 0x3B, 0x3A, 0x39, 0x38,
    0x37, 0x36, 0x35, 0x34, 0x33, 0x2E, 0x31, 0x30, 0x2C, 0x32,
    0x2D, 0x2F, 0x2B, 0x2A, 0x26, 0x28, 0x27, 0x29, 0x25, 0x24,
    0x20, 0x22, 0x21, 0x23, 0x1F, 0x1C, 0x1D, 0x1E, 0x15, 0x1A,
    0x11, 0x18, 0x17, 0x16, 0x1B, 0x14, 0x0D, 0x12, 0x19, 0x0C,
    0x0F, 0x0E, 0x13, 0x10, 0x09, 0x0A, 0x0B, 0x08, 0x07, 0x06,
    0x05, 0x04, 0x03, 0x02, 0x01, 0x00,
};

Byte LengthTable_122[256] =
{
    0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8, 0xF7, 0xF6,
    0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0, 0xEF, 0xEE, 0xED, 0xEC,
    0xEB, 0xEA, 0xE9, 0xE8, 0xE7, 0xE6, 0xE5, 0xE4, 0xE3, 0xE2,
    0xE1, 0xE0, 0xDF, 0xDE, 0xDD, 0xDC, 0xDB, 0xDA, 0xD9, 0xD8,
    0xD7, 0xD6, 0xD5, 0xD4, 0xD3, 0xD2, 0xD1, 0xD0, 0xCF, 0xCE,
    0xCD, 0xCC, 0xCB, 0xCA, 0xC9, 0xC8, 0xC7, 0xC6, 0xC5, 0xC4,
    0xC3, 0xC2, 0xC1, 0xC0, 0xBF, 0xBE, 0xBD, 0xBC, 0xBB, 0xBA,
    0xB9, 0xB8, 0xB7, 0xB6, 0xB5, 0xB4, 0xB3, 0xB2, 0xB1, 0xB0,
    0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8, 0xA7, 0xA6,
    0xA5, 0xA4, 0xA3, 0xA2, 0xA1, 0xA0, 0x9F, 0x9E, 0x9D, 0x9C,
    0x9B, 0x9A, 0x99, 0x98, 0x97, 0x96, 0x95, 0x94, 0x93, 0x92,
    0x91, 0x90, 0x8F, 0x8E, 0x8D, 0x8C, 0x8B, 0x8A, 0x89, 0x88,
    0x87, 0x86, 0x85, 0x84, 0x83, 0x82, 0x81, 0x80, 0x7F, 0x7E,
    0x7D, 0x7C, 0x7B, 0x7A, 0x79, 0x78, 0x77, 0x76, 0x75, 0x74,
    0x73, 0x72, 0x71, 0x70, 0x6F, 0x6E, 0x6D, 0x6C, 0x6B, 0x6A,
    0x69, 0x68, 0x67, 0x66, 0x65, 0x64, 0x63, 0x62, 0x61, 0x60,
    0x5F, 0x5E, 0x5D, 0x5C, 0x5B, 0x5A, 0x59, 0x58, 0x57, 0x56,
    0x55, 0x54, 0x53, 0x52, 0x51, 0x50, 0x4F, 0x4E, 0x4D, 0x4C,
    0x4B, 0x4A, 0x49, 0x03, 0x47, 0x46, 0x45, 0x44, 0x43, 0x42,
    0x41, 0x40, 0x3F, 0x3E, 0x3D, 0x3C, 0x3B, 0x3A, 0x39, 0x38,
    0x37, 0x36, 0x06, 0x34, 0x33, 0x2E, 0x31, 0x30, 0x2C, 0x32,
    0x2D, 0x2F, 0x2B, 0x2A, 0x26, 0x28, 0x27, 0x29, 0x25, 0x24,
    0x20, 0x22, 0x21, 0x23, 0x1F, 0x1C, 0x1D, 0x1E, 0x15, 0x1A,
    0x11, 0x18, 0x17, 0x16, 0x1B, 0x14, 0x0D, 0x12, 0x19, 0x0C,
    0x0F, 0x0E, 0x13, 0x10, 0x09, 0x0A, 0x0B, 0x08, 0x07, 0x35,
    0x05, 0x04, 0x48, 0x02, 0x01, 0x00,
};

UInt32 HashTable[256] = 
{
    0x00000000, 0x77073096, 0xEE0E612C, 0x990951BA, 0x076DC419, 0x706AF48F, 0xE963A535, 0x9E6495A3, 0x0EDB8832, 0x79DCB8A4, 
    0xE0D5E91E, 0x97D2D988, 0x09B64C2B, 0x7EB17CBD, 0xE7B82D07, 0x90BF1D91, 0x1DB71064, 0x6AB020F2, 0xF3B97148, 0x84BE41DE, 
    0x1ADAD47D, 0x6DDDE4EB, 0xF4D4B551, 0x83D385C7, 0x136C9856, 0x646BA8C0, 0xFD62F97A, 0x8A65C9EC, 0x14015C4F, 0x63066CD9, 
    0xFA0F3D63, 0x8D080DF5, 0x3B6E20C8, 0x4C69105E, 0xD56041E4, 0xA2677172, 0x3C03E4D1, 0x4B04D447, 0xD20D85FD, 0xA50AB56B, 
    0x35B5A8FA, 0x42B2986C, 0xDBBBC9D6, 0xACBCF940, 0x32D86CE3, 0x45DF5C75, 0xDCD60DCF, 0xABD13D59, 0x26D930AC, 0x51DE003A, 
    0xC8D75180, 0xBFD06116, 0x21B4F4B5, 0x56B3C423, 0xCFBA9599, 0xB8BDA50F, 0x2802B89E, 0x5F058808, 0xC60CD9B2, 0xB10BE924, 
    0x2F6F7C87, 0x58684C11, 0xC1611DAB, 0xB6662D3D, 0x76DC4190, 0x01DB7106, 0x98D220BC, 0xEFD5102A, 0x71B18589, 0x06B6B51F, 
    0x9FBFE4A5, 0xE8B8D433, 0x7807C9A2, 0x0F00F934, 0x9609A88E, 0xE10E9818, 0x7F6A0DBB, 0x086D3D2D, 0x91646C97, 0xE6635C01, 
    0x6B6B51F4, 0x1C6C6162, 0x856530D8, 0xF262004E, 0x6C0695ED, 0x1B01A57B, 0x8208F4C1, 0xF50FC457, 0x65B0D9C6, 0x12B7E950, 
    0x8BBEB8EA, 0xFCB9887C, 0x62DD1DDF, 0x15DA2D49, 0x8CD37CF3, 0xFBD44C65, 0x4DB26158, 0x3AB551CE, 0xA3BC0074, 0xD4BB30E2, 
    0x4ADFA541, 0x3DD895D7, 0xA4D1C46D, 0xD3D6F4FB, 0x4369E96A, 0x346ED9FC, 0xAD678846, 0xDA60B8D0, 0x44042D73, 0x33031DE5, 
    0xAA0A4C5F, 0xDD0D7CC9, 0x5005713C, 0x270241AA, 0xBE0B1010, 0xC90C2086, 0x5768B525, 0x206F85B3, 0xB966D409, 0xCE61E49F, 
    0x5EDEF90E, 0x29D9C998, 0xB0D09822, 0xC7D7A8B4, 0x59B33D17, 0x2EB40D81, 0xB7BD5C3B, 0xC0BA6CAD, 0xEDB88320, 0x9ABFB3B6, 
    0x03B6E20C, 0x74B1D29A, 0xEAD54739, 0x9DD277AF, 0x04DB2615, 0x73DC1683, 0xE3630B12, 0x94643B84, 0x0D6D6A3E, 0x7A6A5AA8, 
    0xE40ECF0B, 0x9309FF9D, 0x0A00AE27, 0x7D079EB1, 0xF00F9344, 0x8708A3D2, 0x1E01F268, 0x6906C2FE, 0xF762575D, 0x806567CB, 
    0x196C3671, 0x6E6B06E7, 0xFED41B76, 0x89D32BE0, 0x10DA7A5A, 0x67DD4ACC, 0xF9B9DF6F, 0x8EBEEFF9, 0x17B7BE43, 0x60B08ED5, 
    0xD6D6A3E8, 0xA1D1937E, 0x38D8C2C4, 0x4FDFF252, 0xD1BB67F1, 0xA6BC5767, 0x3FB506DD, 0x48B2364B, 0xD80D2BDA, 0xAF0A1B4C, 
    0x36034AF6, 0x41047A60, 0xDF60EFC3, 0xA867DF55, 0x316E8EEF, 0x4669BE79, 0xCB61B38C, 0xBC66831A, 0x256FD2A0, 0x5268E236, 
    0xCC0C7795, 0xBB0B4703, 0x220216B9, 0x5505262F, 0xC5BA3BBE, 0xB2BD0B28, 0x2BB45A92, 0x5CB36A04, 0xC2D7FFA7, 0xB5D0CF31, 
    0x2CD99E8B, 0x5BDEAE1D, 0x9B64C2B0, 0xEC63F226, 0x756AA39C, 0x026D930A, 0x9C0906A9, 0xEB0E363F, 0x72076785, 0x05005713, 
    0x95BF4A82, 0xE2B87A14, 0x7BB12BAE, 0x0CB61B38, 0x92D28E9B, 0xE5D5BE0D, 0x7CDCEFB7, 0x0BDBDF21, 0x86D3D2D4, 0xF1D4E242, 
    0x68DDB3F8, 0x1FDA836E, 0x81BE16CD, 0xF6B9265B, 0x6FB077E1, 0x18B74777, 0x88085AE6, 0xFF0F6A70, 0x66063BCA, 0x11010B5C, 
    0x8F659EFF, 0xF862AE69, 0x616BFFD3, 0x166CCF45, 0xA00AE278, 0xD70DD2EE, 0x4E048354, 0x3903B3C2, 0xA7672661, 0xD06016F7, 
    0x4969474D, 0x3E6E77DB, 0xAED16A4A, 0xD9D65ADC, 0x40DF0B66, 0x37D83BF0, 0xA9BCAE53, 0xDEBB9EC5, 0x47B2CF7F, 0x30B5FFE9, 
    0xBDBDF21C, 0xCABAC28A, 0x53B39330, 0x24B4A3A6, 0xBAD03605, 0xCDD70693, 0x54DE5729, 0x23D967BF, 0xB3667A2E, 0xC4614AB8, 
    0x5D681B02, 0x2A6F2B94, 0xB40BBE37, 0xC30C8EA1, 0x5A05DF1B, 0x2D02EF8D
};