// import { File, Folder, Users, Star, Trash2, Upload, Search, GripVertical, List, ChevronDown, Plus, LayoutList } from "lucide-react";
// import { Button } from "@/components/ui/button";
// import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
// import { Progress } from "@/components/ui/progress";
// import { Input } from "@/components/ui/input";
// import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
// import { Separator } from "@/components/ui/separator";
// import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
// import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
// import { ModeToggle } from "@/components/mode-toggle";
// import DefaultLayout from "@/layouts/DefaultLayout";

// // --- Mock Data ---
// interface FileItem {
//     name: string;
//     size: string;
//     lastModified: string;
//     type: 'file' | 'folder';
// }

// const files: FileItem[] = [
//     { name: "yelling video.mp4", size: "1.59 GB", lastModified: "Oct 25, 2025", type: 'file' },
//     { name: "Malacca Johor trip.zip", size: "142.1 MB", lastModified: "Oct 24, 2025", type: 'file' },
//     { name: "Penang Trip 2027.zip", size: "54.6 MB", lastModified: "Oct 20, 2025", type: 'file' },
//     { name: "Gaming ESL.zip", size: "390.2 MB", lastModified: "Oct 18, 2025", type: 'file' },
//     { name: "Finalprjt.mp4", size: "235.3 MB", lastModified: "Oct 15, 2025", type: 'file' },
//     // ... more files here
// ];

// // --- Sub Components ---

// const Sidebar = () => (
//     <aside className="hidden w-64 border-r p-4 space-y-4 lg:block flex-shrink-0">
//         <Button className="w-full justify-start space-x-2 bg-primary hover:bg-primary/90 shadow-lg text-primary-foreground">
//             <Plus className="h-5 w-5" />
//             <span>New</span>
//         </Button>
//         <nav className="space-y-1">
//             <Button variant="ghost" className="w-full justify-start space-x-3 bg-gray-100 text-primary font-semibold">
//                 <Folder className="h-5 w-5" />
//                 <span>My Drive</span>
//             </Button>
//             <Button variant="ghost" className="w-full justify-start space-x-3">
//                 <Users className="h-5 w-5" />
//                 <span>Shared with me</span>
//             </Button>
//             <Button variant="ghost" className="w-full justify-start space-x-3">
//                 <Star className="h-5 w-5" />
//                 <span>Starred</span>
//             </Button>
//             <Button variant="ghost" className="w-full justify-start space-x-3">
//                 <Trash2 className="h-5 w-5" />
//                 <span>Trash</span>
//             </Button>
//         </nav>
//         <Separator />
//         <Card className="shadow-none">
//             <CardHeader className="p-4">
//                 <CardTitle className="text-sm font-semibold">Storage Space</CardTitle>
//             </CardHeader>
//             <CardContent className="p-4 pt-0 space-y-2">
//                 <Progress value={54} className="h-2" />
//                 <p className="text-xs text-gray-500">8.61 GB used of 15 GB</p>
//                 <Button variant="outline" size="sm" className="w-full">Buy More Storage</Button>
//             </CardContent>
//         </Card>
//     </aside>
// );

// const Header = () => (
//     <header className="flex h-16 items-center justify-between border-b px-4 lg:px-6 shrink-0">
//         <div className="flex items-center space-x-4">
//             <h1 className="text-xl font-bold">Cloud Drive</h1>
//             <div className="relative w-full max-w-lg hidden md:block">
//                 <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
//                 <Input
//                     type="search"
//                     placeholder="Search files and folders..."
//                     className="w-full pl-10 rounded-full bg-gray-50 border-gray-200"
//                 />
//             </div>
//         </div>
//         <div className="flex items-center space-x-3">
//             <TooltipProvider>
//                 <Tooltip>
//                     <TooltipTrigger asChild>
//                         <Button variant="ghost" size="icon">
//                             <Upload className="h-5 w-5" />
//                         </Button>
//                     </TooltipTrigger>
//                     <TooltipContent>
//                         <p>Upload files</p>
//                     </TooltipContent>
//                 </Tooltip>
//             </TooltipProvider>
//             <DropdownMenu>
//                 <DropdownMenuTrigger asChild>
//                     <Button variant="ghost" size="icon" className="rounded-full">
//                         <LayoutList className="h-5 w-5" />
//                     </Button>
//                 </DropdownMenuTrigger>
//                 <DropdownMenuContent align="end">
//                     <DropdownMenuLabel>View Options</DropdownMenuLabel>
//                     <DropdownMenuSeparator />
//                     <DropdownMenuItem>List View (Selected)</DropdownMenuItem>
//                     <DropdownMenuItem>Grid View</DropdownMenuItem>
//                 </DropdownMenuContent>
//             </DropdownMenu>
//             <Button variant="ghost" size="icon" className="rounded-full">
//                 <GripVertical className="h-5 w-5" />
//             </Button>
//         </div>
//     </header>
// );

// const FileListTable = ({ files }: { files: FileItem[] }) => (
//     <div className="overflow-auto max-h-[calc(100vh-14rem)]">
//         <Table>
//             <TableHeader className="sticky top-0 bg-white/90 backdrop-blur-sm z-10">
//                 <TableRow>
//                     <TableHead className="w-[400px]">
//                         <Button variant="ghost" className="p-0 h-auto font-semibold text-gray-600">
//                             Name <ChevronDown className="ml-1 h-3 w-3" />
//                         </Button>
//                     </TableHead>
//                     <TableHead className="text-right">
//                         <Button variant="ghost" className="p-0 h-auto font-semibold text-gray-600">
//                             Size
//                         </Button>
//                     </TableHead>
//                     <TableHead className="text-right">
//                         <Button variant="ghost" className="p-0 h-auto font-semibold text-gray-600">
//                             Last Modified
//                         </Button>
//                     </TableHead>
//                     <TableHead className="w-[50px]"></TableHead>
//                 </TableRow>
//             </TableHeader>
//             <TableBody>
//                 {files.map((file, index) => (
//                     <TableRow key={index} className="group hover:bg-gray-50 cursor-pointer">
//                         <TableCell className="font-medium flex items-center space-x-3 py-2">
//                             {file.type === 'folder' ? <Folder className="h-5 w-5 text-primary" /> : <File className="h-5 w-5 text-gray-400" />}
//                             <span>{file.name}</span>
//                         </TableCell>
//                         <TableCell className="text-right text-gray-600">{file.size}</TableCell>
//                         <TableCell className="text-right text-gray-600">{file.lastModified}</TableCell>
//                         <TableCell className="text-right">
//                             <DropdownMenu>
//                                 <DropdownMenuTrigger asChild>
//                                     <Button variant="ghost" size="icon" className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity">
//                                         <GripVertical className="h-4 w-4" />
//                                     </Button>
//                                 </DropdownMenuTrigger>
//                                 <DropdownMenuContent align="end">
//                                     <DropdownMenuItem>Open</DropdownMenuItem>
//                                     <DropdownMenuItem>Share</DropdownMenuItem>
//                                     <DropdownMenuSeparator />
//                                     <DropdownMenuItem>Download</DropdownMenuItem>
//                                     <DropdownMenuItem className="text-red-600">Delete</DropdownMenuItem>
//                                 </DropdownMenuContent>
//                             </DropdownMenu>
//                         </TableCell>
//                     </TableRow>
//                 ))}
//             </TableBody>
//         </Table>
//     </div>
// );


// // --- Main Component ---
// export default function HomePage() {
//     return (
//         <DefaultLayout>
//             <div className="flex">


//                 {/* 1. Sidebar */}
//                 {/* <Sidebar /> */}

//                 <div className="flex flex-col flex-1">
//                     {/* 2. Header (Navbar) */}
//                     <Header />

//                     {/* 3. Main Content Area */}
//                     <main className="flex-1 p-6 overflow-auto">
//                         <h2 className="text-xl font-semibold mb-4 text-gray-700">My Drive</h2>

//                         {/* <div className="flex items-center justify-between mb-4">
//                             <div className="flex items-center space-x-4 text-sm text-gray-500">
//                                 <span>All Files</span>
//                                 <Separator orientation="vertical" className="h-4" />
//                                 <span>Type: Any</span>
//                                 <Separator orientation="vertical" className="h-4" />
//                                 <span>Sort By: Name</span>
//                             </div>
//                         </div> */}

//                         {/* 4. File List Table */}
//                         <FileListTable files={files} />
//                     </main>
//                 </div>
//             </div>
//         </DefaultLayout>

//     );
// }