package main

var cursor = []byte{0x89, 0x50, 0x4e, 0x47, 0xd, 0xa, 0x1a, 0xa, 0x0, 0x0, 0x0, 0xd, 0x49, 0x48, 0x44, 0x52, 0x0, 0x0, 0x0, 0x40, 0x0, 0x0, 0x0, 0x40, 0x8, 0x6, 0x0, 0x0, 0x0, 0xaa, 0x69, 0x71, 0xde, 0x0, 0x0, 0x2, 0xeb, 0x7a, 0x54, 0x58, 0x74, 0x52, 0x61, 0x77, 0x20, 0x70, 0x72, 0x6f, 0x66, 0x69, 0x6c, 0x65, 0x20, 0x74, 0x79, 0x70, 0x65, 0x20, 0x65, 0x78, 0x69, 0x66, 0x0, 0x0, 0x78, 0xda, 0xed, 0x97, 0x6d, 0x92, 0xdc, 0x28, 0xc, 0x86, 0xff, 0x73, 0x8a, 0x1c, 0x1, 0x49, 0x8, 0x89, 0xe3, 0x60, 0x3e, 0xaa, 0x72, 0x83, 0x3d, 0x7e, 0x5e, 0xb0, 0xdb, 0xd3, 0x3d, 0x93, 0xdd, 0x24, 0xb5, 0xfb, 0x6b, 0xab, 0x4d, 0xd9, 0x60, 0x81, 0x25, 0xf9, 0x7d, 0x64, 0x7a, 0x26, 0x8c, 0xbf, 0xbe, 0xcf, 0xf0, 0xd, 0x7, 0x95, 0xcc, 0x21, 0xa9, 0x79, 0x2e, 0x39, 0x47, 0x1c, 0xa9, 0xa4, 0xc2, 0x15, 0x3, 0x8f, 0xe7, 0x51, 0xf6, 0x95, 0x62, 0xda, 0xd7, 0x7d, 0xa4, 0x6b, 0xa, 0xf7, 0x2f, 0xf6, 0x70, 0x4f, 0x30, 0x4c, 0x82, 0x5e, 0xce, 0x5b, 0xab, 0xd7, 0xfa, 0xa, 0xbb, 0x7e, 0x3c, 0xf0, 0x88, 0x41, 0xc7, 0xab, 0x3d, 0xf8, 0x35, 0xc3, 0x7e, 0x39, 0xa2, 0xdb, 0xf1, 0x3e, 0x64, 0x45, 0x5e, 0xe3, 0xfe, 0x9c, 0x24, 0xec, 0x7c, 0xda, 0xe9, 0xca, 0x24, 0x94, 0x71, 0xe, 0x72, 0x71, 0x7b, 0x4e, 0xf5, 0xb8, 0x1c, 0xb5, 0x47, 0xca, 0xfe, 0x71, 0xa6, 0x3b, 0xad, 0xeb, 0x75, 0x71, 0x1f, 0x5e, 0xc, 0x6, 0x95, 0xba, 0x22, 0x90, 0x30, 0xf, 0x21, 0x89, 0xfb, 0xea, 0x67, 0x6, 0x72, 0x9e, 0x15, 0x67, 0xc2, 0x95, 0x84, 0xb0, 0x8e, 0xa4, 0xec, 0x71, 0xc, 0xe8, 0x92, 0x3c, 0x32, 0x81, 0x20, 0x2f, 0xaf, 0xf7, 0xe8, 0x63, 0x7c, 0x16, 0xe8, 0x45, 0xe4, 0xc7, 0x28, 0x7c, 0x56, 0xff, 0x1e, 0x7d, 0x12, 0x9f, 0xeb, 0x65, 0x97, 0x4f, 0x5a, 0xe6, 0x4b, 0x23, 0xc, 0x7e, 0x3a, 0x41, 0xfa, 0xc9, 0x2e, 0x77, 0x18, 0x7e, 0xe, 0x2c, 0x77, 0x46, 0xfc, 0x3a, 0x61, 0xf2, 0x70, 0xf5, 0x55, 0xe4, 0x39, 0xbb, 0xcf, 0x39, 0xce, 0xb7, 0xab, 0x29, 0x43, 0xd1, 0x7c, 0x55, 0xd4, 0x16, 0x9b, 0x1e, 0x6e, 0xb0, 0xf0, 0x80, 0xe4, 0xb2, 0x1f, 0xcb, 0x68, 0x86, 0x53, 0x31, 0xb6, 0xdd, 0xa, 0x9a, 0xc7, 0x1a, 0x1b, 0x90, 0xf7, 0xd8, 0xe2, 0x81, 0xd6, 0xa8, 0x10, 0x43, 0xeb, 0x19, 0x28, 0x51, 0xa7, 0x4a, 0x93, 0xc6, 0xee, 0x1b, 0x35, 0xa4, 0x98, 0x78, 0xb0, 0xa1, 0x67, 0x6e, 0x2c, 0xdb, 0xe6, 0x62, 0x5c, 0xb8, 0xc9, 0xe2, 0x94, 0x56, 0xa3, 0xc9, 0x6, 0x62, 0x5d, 0x1c, 0x2c, 0x1b, 0x8f, 0x20, 0x2, 0x33, 0xdf, 0xb9, 0xd0, 0x8e, 0x5b, 0x76, 0xbc, 0x46, 0x8e, 0xc8, 0x9d, 0xb0, 0x94, 0x9, 0xce, 0x8, 0x8f, 0xfc, 0x6d, 0xb, 0xff, 0x34, 0xf9, 0x27, 0x2d, 0xcc, 0xd9, 0x96, 0x44, 0x14, 0xfd, 0xd6, 0xa, 0x79, 0xf1, 0xaa, 0x6b, 0xa4, 0xb1, 0xc8, 0xad, 0x2b, 0x56, 0x1, 0x8, 0xcd, 0x8b, 0x9b, 0x6e, 0x81, 0x1f, 0xed, 0xc2, 0x1f, 0x9f, 0xea, 0x7, 0xa5, 0xa, 0x82, 0xba, 0x65, 0x76, 0xbc, 0x60, 0x8d, 0xc7, 0xe9, 0xe2, 0x50, 0xfa, 0xa8, 0x2d, 0xd9, 0x9c, 0x5, 0xeb, 0x14, 0xfd, 0xf9, 0x9, 0x51, 0xb0, 0x7e, 0x39, 0x80, 0x44, 0x88, 0xad, 0x48, 0x6, 0xc5, 0x9f, 0x28, 0x66, 0x12, 0xa5, 0x4c, 0xd1, 0x98, 0x8d, 0x8, 0x3a, 0x3a, 0x0, 0x55, 0x64, 0xce, 0x92, 0xf8, 0x0, 0x1, 0x52, 0xe5, 0x8e, 0x24, 0x39, 0x89, 0x60, 0x3f, 0x32, 0x76, 0x5e, 0xb1, 0xf1, 0x8c, 0xd1, 0x5e, 0xcb, 0xca, 0x99, 0x97, 0x19, 0x7b, 0x13, 0x40, 0xa8, 0x64, 0x31, 0xb0, 0xc1, 0x37, 0x5, 0x58, 0x29, 0x29, 0xea, 0xc7, 0x92, 0xa3, 0x86, 0xaa, 0x8a, 0x26, 0x55, 0xcd, 0x6a, 0xea, 0x41, 0x8b, 0xd6, 0x2c, 0x39, 0x65, 0xcd, 0x39, 0x5b, 0x5e, 0x9b, 0x5c, 0x35, 0xb1, 0x64, 0x6a, 0xd9, 0xcc, 0xdc, 0x8a, 0x55, 0x17, 0x4f, 0xae, 0x9e, 0xdd, 0xdc, 0xbd, 0x78, 0x2d, 0x5c, 0x4, 0x7b, 0xa0, 0x96, 0x5c, 0xac, 0x78, 0x29, 0xa5, 0x56, 0xe, 0x15, 0x81, 0x2a, 0x7c, 0x55, 0xac, 0xaf, 0xb0, 0x1c, 0x7c, 0xc8, 0x91, 0xe, 0x3d, 0xf2, 0x61, 0x87, 0x1f, 0xe5, 0xa8, 0xd, 0xe5, 0xd3, 0x52, 0xd3, 0x96, 0x9b, 0x35, 0x6f, 0xa5, 0xd5, 0xce, 0x5d, 0x3a, 0xb6, 0x89, 0x9e, 0xbb, 0x75, 0xef, 0xa5, 0xd7, 0x41, 0x61, 0x60, 0xa7, 0x18, 0x69, 0xe8, 0xc8, 0xc3, 0x86, 0x8f, 0x32, 0xea, 0x44, 0xad, 0x4d, 0x99, 0x69, 0xea, 0xcc, 0xd3, 0xa6, 0xcf, 0x32, 0xeb, 0x4d, 0xed, 0xa2, 0xfa, 0xa5, 0xfd, 0x1, 0x35, 0xba, 0xa8, 0xf1, 0x26, 0xb5, 0xd6, 0xd9, 0x4d, 0xd, 0xd6, 0x60, 0xf6, 0x70, 0x41, 0x6b, 0x3b, 0xd1, 0xc5, 0xc, 0xc4, 0x38, 0x11, 0x88, 0xdb, 0x22, 0x80, 0x82, 0xe6, 0xc5, 0x2c, 0x3a, 0xa5, 0xc4, 0x8b, 0xdc, 0x62, 0x16, 0xb, 0xe3, 0xa3, 0x50, 0x46, 0x92, 0xba, 0xd8, 0x84, 0x4e, 0x8b, 0x18, 0x10, 0xa6, 0x41, 0xac, 0x93, 0x6e, 0x76, 0x1f, 0xe4, 0x7e, 0x8b, 0x5b, 0x50, 0xff, 0x2d, 0x6e, 0xfc, 0x2b, 0x72, 0x61, 0xa1, 0xfb, 0x2f, 0xc8, 0x5, 0xa0, 0xfb, 0xca, 0xed, 0x27, 0xd4, 0xfa, 0xfa, 0x9d, 0x6b, 0x9b, 0xd8, 0xf9, 0x15, 0x2e, 0x4d, 0xa3, 0xe0, 0xeb, 0xc3, 0xfc, 0xf0, 0x1a, 0xd8, 0xeb, 0xfa, 0x51, 0xab, 0xff, 0xb6, 0x7f, 0x3b, 0x7a, 0x3b, 0x7a, 0x3b, 0x7a, 0x3b, 0x7a, 0x3b, 0x7a, 0x3b, 0x7a, 0x3b, 0xfa, 0x1f, 0x38, 0x9a, 0xf8, 0xe3, 0x1, 0xff, 0xc4, 0x86, 0x1f, 0xe1, 0x76, 0x9d, 0x73, 0xe0, 0xa8, 0xf2, 0x49, 0x0, 0x0, 0x1, 0x84, 0x69, 0x43, 0x43, 0x50, 0x49, 0x43, 0x43, 0x20, 0x70, 0x72, 0x6f, 0x66, 0x69, 0x6c, 0x65, 0x0, 0x0, 0x78, 0x9c, 0x7d, 0x91, 0x3d, 0x48, 0xc3, 0x40, 0x1c, 0xc5, 0x5f, 0x3f, 0x44, 0xd1, 0x8a, 0x5, 0x3b, 0x88, 0x38, 0x64, 0xa8, 0x4e, 0x16, 0x8a, 0x8a, 0x38, 0x6a, 0x15, 0x8a, 0x50, 0x21, 0xd4, 0xa, 0xad, 0x3a, 0x98, 0x5c, 0xfa, 0x5, 0x4d, 0x1a, 0x92, 0x14, 0x17, 0x47, 0xc1, 0xb5, 0xe0, 0xe0, 0xc7, 0x62, 0xd5, 0xc1, 0xc5, 0x59, 0x57, 0x7, 0x57, 0x41, 0x10, 0xfc, 0x0, 0x71, 0x72, 0x74, 0x52, 0x74, 0x91, 0x12, 0xff, 0x97, 0x14, 0x5a, 0xc4, 0x78, 0x70, 0xdc, 0x8f, 0x77, 0xf7, 0x1e, 0x77, 0xef, 0x0, 0x7f, 0xa3, 0xc2, 0x54, 0x33, 0x18, 0x7, 0x54, 0xcd, 0x32, 0xd2, 0xc9, 0x84, 0x90, 0xcd, 0xad, 0xa, 0xdd, 0xaf, 0x8, 0xa2, 0xf, 0x61, 0xc4, 0x11, 0x96, 0x98, 0xa9, 0xcf, 0x89, 0x62, 0xa, 0x9e, 0xe3, 0xeb, 0x1e, 0x3e, 0xbe, 0xde, 0xc5, 0x78, 0x96, 0xf7, 0xb9, 0x3f, 0x47, 0xbf, 0x92, 0x37, 0x19, 0xe0, 0x13, 0x88, 0x67, 0x99, 0x6e, 0x58, 0xc4, 0x1b, 0xc4, 0xd3, 0x9b, 0x96, 0xce, 0x79, 0x9f, 0x38, 0xc2, 0x4a, 0x92, 0x42, 0x7c, 0x4e, 0x3c, 0x6e, 0xd0, 0x5, 0x89, 0x1f, 0xb9, 0x2e, 0xbb, 0xfc, 0xc6, 0xb9, 0xe8, 0xb0, 0x9f, 0x67, 0x46, 0x8c, 0x4c, 0x7a, 0x9e, 0x38, 0x42, 0x2c, 0x14, 0x3b, 0x58, 0xee, 0x60, 0x56, 0x32, 0x54, 0xe2, 0x29, 0xe2, 0xa8, 0xa2, 0x6a, 0x94, 0xef, 0xcf, 0xba, 0xac, 0x70, 0xde, 0xe2, 0xac, 0x56, 0x6a, 0xac, 0x75, 0x4f, 0xfe, 0xc2, 0x50, 0x5e, 0x5b, 0x59, 0xe6, 0x3a, 0xcd, 0x11, 0x24, 0xb1, 0x88, 0x25, 0x88, 0x10, 0x20, 0xa3, 0x86, 0x32, 0x2a, 0xb0, 0x10, 0xa3, 0x55, 0x23, 0xc5, 0x44, 0x9a, 0xf6, 0x13, 0x1e, 0xfe, 0x61, 0xc7, 0x2f, 0x92, 0x4b, 0x26, 0x57, 0x19, 0x8c, 0x1c, 0xb, 0xa8, 0x42, 0x85, 0xe4, 0xf8, 0xc1, 0xff, 0xe0, 0x77, 0xb7, 0x66, 0x61, 0x72, 0xc2, 0x4d, 0xa, 0x25, 0x80, 0xae, 0x17, 0xdb, 0xfe, 0x18, 0x5, 0xba, 0x77, 0x81, 0x66, 0xdd, 0xb6, 0xbf, 0x8f, 0x6d, 0xbb, 0x79, 0x2, 0x4, 0x9e, 0x81, 0x2b, 0xad, 0xed, 0xaf, 0x36, 0x80, 0x99, 0x4f, 0xd2, 0xeb, 0x6d, 0x2d, 0x7a, 0x4, 0xc, 0x6c, 0x3, 0x17, 0xd7, 0x6d, 0x4d, 0xde, 0x3, 0x2e, 0x77, 0x80, 0xa1, 0x27, 0x5d, 0x32, 0x24, 0x47, 0xa, 0xd0, 0xf4, 0x17, 0xa, 0xc0, 0xfb, 0x19, 0x7d, 0x53, 0xe, 0x18, 0xbc, 0x5, 0x7a, 0xd7, 0xdc, 0xde, 0x5a, 0xfb, 0x38, 0x7d, 0x0, 0x32, 0xd4, 0x55, 0xea, 0x6, 0x38, 0x38, 0x4, 0xc6, 0x8a, 0x94, 0xbd, 0xee, 0xf1, 0xee, 0x9e, 0xce, 0xde, 0xfe, 0x3d, 0xd3, 0xea, 0xef, 0x7, 0x57, 0x84, 0x72, 0x9c, 0x83, 0xaf, 0xe1, 0x29, 0x0, 0x0, 0x0, 0x9, 0x70, 0x48, 0x59, 0x73, 0x0, 0x0, 0xb, 0x13, 0x0, 0x0, 0xb, 0x13, 0x1, 0x0, 0x9a, 0x9c, 0x18, 0x0, 0x0, 0x0, 0x7, 0x74, 0x49, 0x4d, 0x45, 0x7, 0xe4, 0x4, 0xb, 0x11, 0xa, 0x2c, 0xb6, 0x53, 0x9f, 0xdb, 0x0, 0x0, 0x1, 0xbc, 0x49, 0x44, 0x41, 0x54, 0x78, 0xda, 0xed, 0x98, 0x3b, 0x2f, 0x44, 0x41, 0x14, 0x80, 0xbf, 0x65, 0x57, 0x54, 0x44, 0xfc, 0x0, 0xd5, 0x26, 0x2a, 0x89, 0x56, 0x25, 0xfc, 0x1, 0xbf, 0x41, 0x21, 0xa1, 0x53, 0x8, 0x4a, 0x9d, 0x4a, 0xa2, 0x42, 0x8d, 0x90, 0x58, 0x14, 0x58, 0x85, 0x47, 0x22, 0x91, 0x6c, 0x88, 0x4, 0x9d, 0x57, 0x34, 0x1e, 0xd, 0x21, 0x8b, 0x66, 0x25, 0x5c, 0xcd, 0x99, 0x64, 0x72, 0xb3, 0x76, 0x37, 0xd1, 0xb8, 0x73, 0xce, 0x97, 0x4c, 0x31, 0x99, 0xd3, 0x9c, 0x6f, 0xce, 0xdc, 0x33, 0x73, 0x1, 0x46, 0x50, 0xce, 0x23, 0xb0, 0xac, 0x59, 0xc0, 0x3d, 0x10, 0x1, 0x39, 0xed, 0x2, 0x22, 0x60, 0x41, 0xbb, 0x80, 0x8, 0xd8, 0xd2, 0x2e, 0x20, 0x2, 0x56, 0x81, 0x94, 0x36, 0x1, 0x45, 0xe0, 0xd4, 0x93, 0x30, 0xaf, 0x4d, 0xc0, 0x93, 0xcc, 0x77, 0xb5, 0x1d, 0x87, 0xb8, 0x80, 0xc, 0x70, 0xe4, 0x49, 0x58, 0xc, 0xfd, 0x38, 0xc4, 0x5, 0x38, 0x7c, 0x9, 0x2b, 0x1a, 0x5, 0xa4, 0x81, 0xbc, 0x27, 0x61, 0x5b, 0x9b, 0x0, 0x80, 0x7a, 0xe0, 0x38, 0x76, 0x4f, 0x48, 0x69, 0x12, 0xe0, 0xd8, 0xf7, 0x24, 0x6c, 0x6a, 0x14, 0x90, 0x96, 0x8e, 0xe0, 0x24, 0xe4, 0x43, 0xaa, 0x84, 0x5a, 0x4, 0x20, 0x9, 0xfb, 0x1f, 0xc6, 0x25, 0x6d, 0x2, 0x1c, 0x9b, 0x9e, 0x84, 0x5d, 0x8d, 0x2, 0xea, 0x63, 0xdd, 0x61, 0x3, 0xa8, 0xd3, 0x24, 0xc0, 0x71, 0xe8, 0x49, 0xc8, 0x69, 0x14, 0x80, 0x5c, 0x90, 0x9c, 0x84, 0x3, 0x8d, 0x2, 0x0, 0xd6, 0x3c, 0x9, 0xeb, 0x49, 0xec, 0xe, 0xb5, 0x8, 0xa8, 0x3, 0x1a, 0x81, 0x26, 0xa0, 0xb, 0x18, 0x4, 0x66, 0x80, 0x13, 0xe0, 0x33, 0xe9, 0xff, 0x13, 0x7e, 0x13, 0x90, 0xf2, 0x76, 0xb3, 0xa5, 0xcc, 0x3f, 0x83, 0x4a, 0xe3, 0xc, 0x68, 0x48, 0x42, 0xf2, 0xe9, 0xa, 0x6b, 0x6d, 0x40, 0x1, 0x98, 0x0, 0x66, 0x81, 0x49, 0x60, 0x4c, 0xd6, 0x8a, 0x22, 0xee, 0x15, 0x78, 0x6, 0x6e, 0x80, 0xb, 0x19, 0x57, 0x7f, 0x38, 0x4e, 0xff, 0xa6, 0x2, 0xb2, 0x40, 0xc9, 0xdb, 0xd1, 0x66, 0xa0, 0x15, 0x78, 0x93, 0xf9, 0x75, 0xc8, 0x57, 0xe1, 0xa1, 0x32, 0x25, 0x3d, 0x2e, 0x6b, 0x73, 0x32, 0xff, 0x6, 0x3a, 0x42, 0x13, 0xf0, 0x2, 0x4c, 0x79, 0x49, 0xbf, 0xcb, 0x4e, 0x47, 0xb2, 0xf3, 0xe, 0x57, 0x19, 0x7b, 0xa1, 0x9, 0x88, 0x8f, 0x2c, 0xd0, 0x9, 0x7c, 0xc9, 0x7c, 0x5a, 0xe2, 0x47, 0xbd, 0x98, 0xde, 0x10, 0x5, 0x9c, 0xcb, 0x99, 0x77, 0x5c, 0x7a, 0x15, 0x91, 0x2, 0x6, 0xbc, 0xd8, 0x42, 0x8, 0xaf, 0xc2, 0xfb, 0x2a, 0x3d, 0xbc, 0xdd, 0x5b, 0xff, 0x88, 0xc9, 0x7a, 0x90, 0xb7, 0x41, 0x10, 0x2, 0x86, 0x2b, 0xc4, 0xec, 0xc4, 0x12, 0x2f, 0x1, 0x3d, 0x49, 0xe9, 0xf5, 0xd5, 0xb8, 0x3, 0xfa, 0xaa, 0xc4, 0x74, 0x4b, 0xe2, 0xb7, 0xd2, 0x25, 0x82, 0x22, 0x53, 0x63, 0x5c, 0x3f, 0x86, 0x61, 0x18, 0x86, 0x61, 0x18, 0x86, 0x61, 0x18, 0x86, 0x61, 0x18, 0x86, 0x61, 0x18, 0x86, 0x61, 0x18, 0x86, 0x61, 0x18, 0x86, 0x61, 0x18, 0x9, 0xe4, 0x7, 0xc1, 0xdd, 0xd8, 0x1d, 0x58, 0xff, 0xf9, 0xa2, 0x0, 0x0, 0x0, 0x0, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82}
