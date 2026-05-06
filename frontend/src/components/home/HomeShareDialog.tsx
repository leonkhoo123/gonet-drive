import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { createShare, isNeverExpires } from "@/api/api-share";
import type { CreateShareResponse, CreateShareRequest } from "@/api/api-share";
import { Loader2, Copy, Check } from "lucide-react";
import { toast } from "sonner";
import { AxiosError } from "axios";

interface HomeShareDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  itemPath: string | null;
}

export default function HomeShareDialog({
  isOpen,
  onOpenChange,
  itemPath
}: HomeShareDialogProps) {
  const [loading, setLoading] = useState(false);
  const [successData, setSuccessData] = useState<CreateShareResponse | null>(null);
  
  // Form state
  const [description, setDescription] = useState<string>("");
  const [expiresInHours, setExpiresInHours] = useState<string>("168"); // Default 7 days
  const [authority, setAuthority] = useState<"view" | "modify">("view");
  
  // Copy states
  const [copiedLink, setCopiedLink] = useState(false);

  // Reset state when dialog opens with a new item
  useEffect(() => {
    if (isOpen) {
      setLoading(false);
      setSuccessData(null);
      setDescription("");
      setExpiresInHours("168");
      setAuthority("view");
      setCopiedLink(false);
    }
  }, [isOpen, itemPath]);

  const handleShare = async () => {
    if (!itemPath || !description.trim()) {
      if (!description.trim()) {
        toast.error("Description is required");
      }
      return;
    }

    setLoading(true);
    try {
      const req: CreateShareRequest = {
        path: itemPath,
        description: description.trim(),
        expires_in_hours: parseInt(expiresInHours, 10),
        authority: authority,
      };
      
      const response = await createShare(req);
      setSuccessData(response);
      toast.success("Share created successfully");
    } catch (error: unknown) {
      console.error("Failed to create share:", error);
      const err = error as AxiosError<{ error: string }>;
      toast.error(err.response?.data?.error ?? "Failed to create share");
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = async () => {
    if (!successData) return;
    const shareLink = `${window.location.origin}/share/${successData.share.id}`;
    const never = isNeverExpires(successData.share.expires_at);
    const expiryDate = never ? "Never Expires" : new Date(successData.share.expires_at).toLocaleString();
    const textToCopy = `Please use the PIN below to access the shared content.\nLink: ${shareLink}\nPIN: ${successData.pin}\nExpires: ${expiryDate}`;
    
    try {
      await navigator.clipboard.writeText(textToCopy);
      setCopiedLink(true);
      setTimeout(() => { setCopiedLink(false); }, 2000);
      toast.success("Share info copied to clipboard");
    } catch (err) {
      console.error(err);
      toast.error("Failed to copy to clipboard");
    }
  };

  const itemName = itemPath ? itemPath.split('/').pop() : '';

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{successData ? "Share Created" : "Share Item"}</DialogTitle>
          <DialogDescription>
            {successData ? "Your item is now successfully shared." : `Share ${itemName}`}
          </DialogDescription>
        </DialogHeader>

        {successData ? (() => {
          const never = isNeverExpires(successData.share.expires_at);
          return (
          <div className="grid gap-6 py-4">
            <div className="flex flex-col gap-2">
              <Label>Share Information</Label>
              <div className="relative">
                <div className="w-full bg-muted p-3 pr-12 rounded-md text-sm font-mono border overflow-x-auto whitespace-pre-wrap break-all">
                  Please use the PIN below to access the shared content.
                  <br />
                  <br />
                  Link: {`${window.location.origin}/share/${successData.share.id}`}
                  <br />
                  <br />
                  PIN: <span className="font-bold tracking-widest">{successData.pin}</span>
                  {!never && (
                    <>
                      <br />
                      <br />
                      Expires: {new Date(successData.share.expires_at).toLocaleString()}
                    </>
                  )}
                </div>
                <Button 
                  size="icon" 
                  variant="outline" 
                  onClick={handleCopy}
                  title="Copy Share Info"
                  className="absolute top-2 right-2 h-8 w-8 shrink-0"
                >
                  {copiedLink ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
                </Button>
              </div>
            </div>
            
            <p className="text-sm text-muted-foreground">
              Anyone with this link and PIN can {successData.share.authority === 'modify' ? 'view and modify' : 'view'} this item{never ? " indefinitely" : " until it expires"}.
            </p>
          </div>
          );
        })() : (
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label>Description <span className="text-red-500">*</span></Label>
              <Input
                placeholder="What is this share for?"
                value={description}
                onChange={(e) => { setDescription(e.target.value); }}
                disabled={loading}
              />
            </div>

            <div className="grid gap-2">
              <Label>Expiration Time</Label>
              <select
                value={expiresInHours}
                onChange={(e) => { setExpiresInHours(e.target.value); }}
                disabled={loading}
                className="flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm ring-offset-background focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
              >
                <option value="24" className="bg-background text-foreground">1 Day</option>
                <option value="168" className="bg-background text-foreground">7 Days (Default)</option>
                <option value="336" className="bg-background text-foreground">14 Days</option>
                <option value="720" className="bg-background text-foreground">1 Month (30 Days)</option>
                <option value="-1" className="bg-background text-foreground">Never Expire</option>
              </select>
            </div>

            <div className="grid gap-2">
              <Label>Permissions</Label>
              <select 
                value={authority} 
                onChange={(e) => { setAuthority(e.target.value as "view" | "modify"); }}
                disabled={loading}
                className="flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm ring-offset-background focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
              >
                <option value="view" className="bg-background text-foreground">View Only</option>
                <option value="modify" className="bg-background text-foreground">Allow Modification</option>
              </select>
            </div>
          </div>
        )}

        <DialogFooter>
          {successData ? (
            <Button onClick={() => { onOpenChange(false); }}>Close</Button>
          ) : (
            <>
              <Button variant="outline" onClick={() => { onOpenChange(false); }} disabled={loading}>
                Cancel
              </Button>
              <Button onClick={handleShare} disabled={loading || !itemPath}>
                {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Create Share
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
