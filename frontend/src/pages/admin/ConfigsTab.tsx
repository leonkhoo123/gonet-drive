import { useRef, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Switch } from "@/components/ui/switch";
import { toast } from 'sonner';
import axiosLayer from '@/api/axiosLayer';
import { getConfig } from '@/config';
import { Logo } from '@/components/Logo';
import type { ConfigItem } from "@/api/api-config";
import { Loader2, Settings2 } from "lucide-react";

interface ConfigsTabProps {
  configs: ConfigItem[];
  loadingConfigs: boolean;
  savingConfigs: boolean;
  hasConfigChanges: boolean;
  onUpdateLocalConfig: (id: number, field: keyof ConfigItem, value: string | boolean | null) => void;
  onSave: () => void;
}

const ConfigsTab = ({
  configs,
  loadingConfigs,
  savingConfigs,
  hasConfigChanges,
  onUpdateLocalConfig,
  onSave,
}: ConfigsTabProps) => {
  const [uploadingLogo, setUploadingLogo] = useState(false);
  const [logoUrl, setLogoUrl] = useState(() => `${getConfig().apiBaseUrl}/config/logo`);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleLogoUpload = async () => {
    if (!selectedFile) return;

    try {
      setUploadingLogo(true);
      const formData = new FormData();
      formData.append('logo', selectedFile);

      await axiosLayer.put('/user/admin/config/logo', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      });

      toast.success('Logo updated successfully');
      setLogoUrl(`${getConfig().apiBaseUrl}/config/logo?t=${String(Date.now())}`);
      setSelectedFile(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    } catch (err) {
      const errorMsg = (err as { response?: { data?: { error?: string } } }).response?.data?.error ?? 'Failed to update logo';
      toast.error(errorMsg);
    } finally {
      setUploadingLogo(false);
    }
  };

  const handleCancelLogo = () => {
    setSelectedFile(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  return (
    <div className="space-y-4">
      <Card className="flex flex-col border-border shadow-sm py-3 md:py-6 gap-3 md:gap-6">
        <CardHeader className="bg-muted/30 pb-3 px-4 md:px-6">
          <CardTitle>Site Logo</CardTitle>
          <CardDescription className="hidden sm:block">Upload a custom logo for your cloud instance. Recommended 1:1 aspect ratio, PNG only.</CardDescription>
        </CardHeader>
        <CardContent className="pt-0 md:pt-6 px-4 md:px-6 flex flex-row flex-wrap items-center gap-4 sm:gap-6">
          <div className="bg-muted rounded-lg p-4 flex items-center justify-center border w-32 h-32 shrink-0">
            <Logo src={logoUrl} className="max-w-full max-h-full object-contain" />
          </div>
          <div className="space-y-4 flex-1">
            <div className="flex flex-col sm:flex-row sm:items-center gap-4">
              <Label className="text-sm font-medium">Choose file to upload :</Label>
              <input
                type="file"
                accept="image/png"
                className="hidden"
                ref={fileInputRef}
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) {
                    if (file.type !== 'image/png') {
                      toast.error('Only PNG files are allowed');
                      e.target.value = '';
                      return;
                    }
                    if (file.size > 5 * 1024 * 1024) {
                      toast.error('File size exceeds 5MB limit');
                      e.target.value = '';
                      return;
                    }
                    setSelectedFile(file);
                  }
                }}
              />

              {!selectedFile ? (
                <Button
                  variant="outline"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={uploadingLogo}
                >
                  Browse
                </Button>
              ) : (
                <div className="flex items-center gap-2 flex-1 w-full max-w-md">
                  <Input
                    value={selectedFile.name}
                    readOnly
                    className="flex-1 text-muted-foreground bg-muted/50 truncate"
                  />
                  <Button
                    onClick={() => { void handleLogoUpload(); }}
                    disabled={uploadingLogo}
                  >
                    {uploadingLogo && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    Upload
                  </Button>
                  <Button
                    variant="outline"
                    onClick={handleCancelLogo}
                    disabled={uploadingLogo}
                  >
                    Cancel
                  </Button>
                </div>
              )}
            </div>
            {uploadingLogo && <p className="text-sm text-muted-foreground flex items-center"><Loader2 className="mr-2 h-4 w-4 animate-spin" /> Uploading...</p>}
          </div>
        </CardContent>
      </Card>

      <Card className="flex flex-col border-border shadow-sm py-3 md:py-6 gap-3 md:gap-6">
        <CardHeader className="flex flex-row items-center justify-between bg-muted/30 pb-3 px-4 md:px-6">
          <div>
            <CardTitle>System Configurations</CardTitle>
            <CardDescription className="hidden sm:block">Manage global settings and features for your cloud instance.</CardDescription>
          </div>
          <Button
            onClick={onSave}
            disabled={!hasConfigChanges || savingConfigs || loadingConfigs}
            className="bg-primary hover:bg-primary/90 text-primary-foreground shadow-sm"
          >
            {savingConfigs && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Save Changes
          </Button>
        </CardHeader>
        <CardContent className="pt-0 md:pt-6 px-4 md:px-6">
          {loadingConfigs && configs.length === 0 ? (
            <div className="flex items-center justify-center p-12">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
            </div>
          ) : configs.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground rounded-md border">
              <Settings2 className="h-8 w-8 mx-auto mb-3 opacity-20" />
              No configurations found
            </div>
          ) : (
            <>
              <div className="hidden md:block rounded-md border overflow-hidden">
                <Table>
                  <TableHeader className="bg-muted/50">
                    <TableRow>
                      <TableHead className="font-semibold">Configuration</TableHead>
                      <TableHead className="font-semibold">Value</TableHead>
                      <TableHead className="w-[120px] font-semibold text-center">Enabled</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {configs.map((config) => (
                      <TableRow key={config.id} className="hover:bg-muted/30 transition-colors">
                        <TableCell className="py-3">
                          <div className="font-medium text-base">{config.config_name}</div>
                          <div className="text-xs text-muted-foreground mt-1 flex items-center gap-2">
                            <span className="bg-muted px-2 py-0.5 rounded-full font-mono">{config.config_type}</span>
                            {config.config_unit && (
                              <span className="text-muted-foreground/80">Unit: {config.config_unit}</span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell className="py-3">
                          <Input
                            value={config.config_value ?? ""}
                            onChange={(e) => {
                              onUpdateLocalConfig(config.id, "config_value", e.target.value);
                            }}
                            placeholder="Value"
                            disabled={savingConfigs}
                            className="focus-visible:ring-ring max-w-md"
                          />
                        </TableCell>
                        <TableCell className="py-3">
                          <div className="flex justify-center">
                            <Switch
                              checked={config.is_enabled}
                              onCheckedChange={(checked) => {
                                onUpdateLocalConfig(config.id, "is_enabled", checked);
                              }}
                              disabled={savingConfigs}
                              className="data-[state=checked]:bg-primary"
                            />
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <div className="md:hidden space-y-2">
                {configs.map((config) => (
                  <div key={config.id} className="rounded-md border p-3 space-y-2 bg-card">
                    <div>
                      <div className="font-medium text-base">{config.config_name}</div>
                      <div className="text-xs text-muted-foreground mt-1 flex items-center gap-2">
                        <span className="bg-muted px-2 py-0.5 rounded-full font-mono">{config.config_type}</span>
                        {config.config_unit && (
                          <span className="text-muted-foreground/80">Unit: {config.config_unit}</span>
                        )}
                      </div>
                    </div>
                    <Input
                      value={config.config_value ?? ""}
                      onChange={(e) => {
                        onUpdateLocalConfig(config.id, "config_value", e.target.value);
                      }}
                      placeholder="Value"
                      disabled={savingConfigs}
                      className="focus-visible:ring-ring"
                    />
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-muted-foreground">Enabled</span>
                      <Switch
                        checked={config.is_enabled}
                        onCheckedChange={(checked) => {
                          onUpdateLocalConfig(config.id, "is_enabled", checked);
                        }}
                        disabled={savingConfigs}
                        className="data-[state=checked]:bg-primary"
                      />
                    </div>
                  </div>
                ))}
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

export default ConfigsTab;
