"use client";

import { use, useCallback, useEffect, useState } from "react";
import { Plus } from "lucide-react";

import { api } from "@/lib/api/client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type MemberRole = "owner" | "editor" | "viewer";

interface Member {
  id: string;
  workspace_id: string;
  name: string;
  email: string;
  role: MemberRole;
}

const roleColors: Record<MemberRole, "default" | "secondary" | "outline"> = {
  owner: "default",
  editor: "secondary",
  viewer: "outline",
};

export default function MembersPage({
  params,
}: {
  params: Promise<{ wsId: string }>;
}) {
  const { wsId } = use(params);
  const [members, setMembers] = useState<Member[]>([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [editMember, setEditMember] = useState<Member | null>(null);
  const [form, setForm] = useState({
    name: "",
    email: "",
    role: "editor" as MemberRole,
  });

  const fetchMembers = useCallback(async () => {
    const { data } = await api.GET("/workspaces/{wsId}/members", {
      params: { path: { wsId } },
    });
    if (data) {
      setMembers(data);
    }
    setLoading(false);
  }, [wsId]);

  useEffect(() => {
    void fetchMembers();
  }, [fetchMembers]);

  const resetForm = () => {
    setForm({ name: "", email: "", role: "editor" });
    setEditMember(null);
  };

  const handleOpenCreate = () => {
    resetForm();
    setOpen(true);
  };

  const handleOpenEdit = (member: Member) => {
    setEditMember(member);
    setForm({
      name: member.name,
      email: member.email,
      role: member.role,
    });
    setOpen(true);
  };

  const handleSave = async () => {
    if (editMember) {
      const { data } = await api.PUT("/workspaces/{wsId}/members/{id}", {
        params: { path: { wsId, id: editMember.id } },
        body: form,
      });
      if (data) {
        setMembers((prev) =>
          prev.map((m) => (m.id === data.id ? data : m)),
        );
      }
    } else {
      const { data } = await api.POST("/workspaces/{wsId}/members", {
        params: { path: { wsId } },
        body: form,
      });
      if (data) {
        setMembers((prev) => [...prev, data]);
      }
    }
    setOpen(false);
    resetForm();
  };

  const handleDelete = async (id: string) => {
    await api.DELETE("/workspaces/{wsId}/members/{id}", {
      params: { path: { wsId, id } },
    });
    setMembers((prev) => prev.filter((m) => m.id !== id));
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Members</h1>
        <Dialog
          open={open}
          onOpenChange={(v) => {
            setOpen(v);
            if (!v) resetForm();
          }}
        >
          <DialogTrigger asChild>
            <Button onClick={handleOpenCreate}>
              <Plus className="size-4" />
              Add Member
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                {editMember ? "Edit Member" : "Add Member"}
              </DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label>Name</Label>
                <Input
                  value={form.name}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, name: e.target.value }))
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label>Email</Label>
                <Input
                  type="email"
                  value={form.email}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, email: e.target.value }))
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label>Role</Label>
                <Select
                  value={form.role}
                  onValueChange={(v: MemberRole) =>
                    setForm((f) => ({ ...f, role: v }))
                  }
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="owner">Owner</SelectItem>
                    <SelectItem value="editor">Editor</SelectItem>
                    <SelectItem value="viewer">Viewer</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <DialogFooter>
              <Button
                onClick={() => void handleSave()}
                disabled={!form.name || !form.email}
              >
                {editMember ? "Save" : "Add"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {loading ? (
        <p className="text-muted-foreground">Loading...</p>
      ) : members.length === 0 ? (
        <p className="text-muted-foreground">No members yet.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Role</TableHead>
              <TableHead className="w-32">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {members.map((member) => (
              <TableRow key={member.id}>
                <TableCell className="font-medium">{member.name}</TableCell>
                <TableCell>{member.email}</TableCell>
                <TableCell>
                  <Badge variant={roleColors[member.role]}>
                    {member.role}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleOpenEdit(member)}
                    >
                      Edit
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => void handleDelete(member.id)}
                    >
                      Delete
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
